package sqldb

import (
	"context"
	"errors"
	"fmt"
	"time"

	corespdb "github.com/bnb-chain/greenfield-storage-provider/core/spdb"
	"github.com/bnb-chain/greenfield-storage-provider/pkg/log"
	"github.com/bnb-chain/greenfield-storage-provider/pkg/metrics"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	// SPDBSuccessCheckQuotaAndAddReadRecord defines the metrics label of successfully check and add read record
	SPDBSuccessCheckQuotaAndAddReadRecord = "check_and_add_read_record_success"
	// SPDBFailureCheckQuotaAndAddReadRecord defines the metrics label of unsuccessfully check and add read record
	SPDBFailureCheckQuotaAndAddReadRecord = "check_and_add_read_record_failure"
	// SPDBSuccessGetBucketTraffic defines the metrics label of successfully get bucket traffic
	SPDBSuccessGetBucketTraffic = "get_bucket_traffic_success"
	// SPDBFailureGetBucketTraffic defines the metrics label of unsuccessfully get bucket traffic
	SPDBFailureGetBucketTraffic = "get_bucket_traffic_failure"
	// SPDBSuccessGetReadRecord defines the metrics label of successfully get read record
	SPDBSuccessGetReadRecord = "get_read_record_success"
	// SPDBFailureGetReadRecord defines the metrics label of unsuccessfully get read record
	SPDBFailureGetReadRecord = "get_read_record_failure"
	// SPDBSuccessGetBucketReadRecord defines the metrics label of successfully get bucket read record
	SPDBSuccessGetBucketReadRecord = "get_bucket_read_record_success"
	// SPDBFailureGetBucketReadRecord defines the metrics label of unsuccessfully get bucket read record
	SPDBFailureGetBucketReadRecord = "get_bucket_read_record_failure"
	// SPDBSuccessGetObjectReadRecord defines the metrics label of successfully get object read record
	SPDBSuccessGetObjectReadRecord = "get_object_read_record_success"
	// SPDBFailureGetObjectReadRecord defines the metrics label of unsuccessfully get object read record
	SPDBFailureGetObjectReadRecord = "get_object_read_record_failure"
	// SPDBSuccessGetUserReadRecord defines the metrics label of successfully get user read record
	SPDBSuccessGetUserReadRecord = "get_user_read_record_success"
	// SPDBFailureGetUserReadRecord defines the metrics label of unsuccessfully get user read record
	SPDBFailureGetUserReadRecord = "get_user_read_record_failure"
)

// CheckQuotaAndAddReadRecord check current quota, and add read record
func (s *SpDBImpl) CheckQuotaAndAddReadRecord(record *corespdb.ReadRecord, quota *corespdb.BucketQuota) (err error) {
	startTime := time.Now()
	defer func() {
		if err != nil {
			metrics.SPDBCounter.WithLabelValues(SPDBFailureCheckQuotaAndAddReadRecord).Inc()
			metrics.SPDBTime.WithLabelValues(SPDBFailureCheckQuotaAndAddReadRecord).Observe(
				time.Since(startTime).Seconds())
			return
		}
		metrics.SPDBCounter.WithLabelValues(SPDBSuccessCheckQuotaAndAddReadRecord).Inc()
		metrics.SPDBTime.WithLabelValues(SPDBSuccessCheckQuotaAndAddReadRecord).Observe(
			time.Since(startTime).Seconds())
	}()

	err = s.updateConsumedQuota(record, quota)
	if err != nil {
		log.Errorw("failed to commit the transaction of updating bucketTraffic table, ", "error", err)
		return err
	}

	// add read record
	insertReadRecord := &ReadRecordTable{
		BucketID:        record.BucketID,
		ObjectID:        record.ObjectID,
		UserAddress:     record.UserAddress,
		ReadTimestampUs: record.ReadTimestampUs,
		BucketName:      record.BucketName,
		ObjectName:      record.ObjectName,
		ReadSize:        record.ReadSize,
	}
	result := s.db.Create(insertReadRecord)
	if result.Error != nil || result.RowsAffected != 1 {
		err = fmt.Errorf("failed to insert read record table: %s", result.Error)
		return err
	}
	return nil
}

func getUpdatedConsumedQuota(record *corespdb.ReadRecord, freeQuota, freeConsumedQuota, totalConsumeQuota, chargedQuota uint64) (uint64, uint64, error) {
	recordQuotaCost := record.ReadSize
	needCheckChainQuota := true
	freeQuotaRemain := freeQuota - freeConsumedQuota
	// if remain free quota more than 0, consume free quota first
	if freeQuotaRemain > 0 && recordQuotaCost < freeQuotaRemain {
		// if free quota is enough, no need to check charged quota
		totalConsumeQuota += recordQuotaCost
		freeConsumedQuota += recordQuotaCost
		needCheckChainQuota = false
	}
	// if free quota is not enough, check the charged quota
	if needCheckChainQuota {
		// the quota size of this month should be (chargedQuota + freeQuotaRemain)
		if totalConsumeQuota+recordQuotaCost > chargedQuota+freeQuotaRemain {
			return 0, 0, ErrCheckQuotaEnough
		}
		totalConsumeQuota += recordQuotaCost
		if freeQuotaRemain > 0 {
			freeConsumedQuota += freeQuotaRemain
		}
	}

	return freeConsumedQuota, totalConsumeQuota, nil
}

// updateConsumedQuota update the consumed quota of BucketTraffic table in the transaction way
func (s *SpDBImpl) updateConsumedQuota(record *corespdb.ReadRecord, quota *corespdb.BucketQuota) error {
	yearMonth := TimeToYearMonth(TimestampUsToTime(record.ReadTimestampUs))
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var bucketTraffic BucketTrafficTable
		var err error
		if err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("bucket_id = ? and month = ?", record.BucketID, yearMonth).Find(&bucketTraffic).Error; err != nil {
			return fmt.Errorf("failed to query bucket traffic table: %v", err)
		}

		// if charged quota changed, update the new value
		if bucketTraffic.ChargedQuotaSize != quota.ChargedQuotaSize {
			result := tx.Model(&bucketTraffic).
				Updates(BucketTrafficTable{
					ChargedQuotaSize: quota.ChargedQuotaSize,
					ModifiedTime:     time.Now(),
				})
			if result.Error != nil {
				return fmt.Errorf("failed to update bucket traffic table: %s", result.Error)
			}

			if result.RowsAffected != 1 {
				return fmt.Errorf("update traffic of %s has affected more than one rows %d, "+
					"update charged quota %d", bucketTraffic.BucketName, result.RowsAffected, quota.ChargedQuotaSize)
			}
		}

		// compute the new consumed quota size to be updated
		updatedReadConsumedSize, updatedFreeConsumedSize, err := getUpdatedConsumedQuota(record,
			bucketTraffic.FreeQuotaSize, bucketTraffic.FreeQuotaConsumedSize,
			bucketTraffic.ReadConsumedSize, bucketTraffic.ChargedQuotaSize)
		if err != nil {
			return err
		}

		if err = tx.Model(&bucketTraffic).
			Updates(BucketTrafficTable{
				ReadConsumedSize:      updatedReadConsumedSize,
				FreeQuotaConsumedSize: updatedFreeConsumedSize,
				ModifiedTime:          time.Now(),
			}).Error; err != nil {
			return fmt.Errorf("failed to update bucket traffic table: %v", err)
		}

		return nil
	})

	return err
}

// InitBucketTraffic init the bucket traffic table
func (s *SpDBImpl) InitBucketTraffic(record *corespdb.ReadRecord, quota *corespdb.BucketQuota) error {
	bucketID := record.BucketID
	bucketName := record.BucketName
	yearMonth := TimestampYearMonth(record.ReadTimestampUs)
	// if not created, init the bucket id in transaction
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var insertBucketTraffic *BucketTrafficTable
		var bucketTraffic BucketTrafficTable
		result := s.db.Where("bucket_id = ?", bucketID).First(&bucketTraffic)
		if result.Error != nil {
			if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return result.Error
			} else {
				// If the record of this bucket id does not exist, then the free quota consumed is initialized to 0
				insertBucketTraffic = &BucketTrafficTable{
					BucketID:              bucketID,
					Month:                 yearMonth,
					FreeQuotaSize:         quota.FreeQuotaSize,
					FreeQuotaConsumedSize: 0,
					BucketName:            bucketName,
					ReadConsumedSize:      0,
					ChargedQuotaSize:      quota.ChargedQuotaSize,
					ModifiedTime:          time.Now(),
				}
			}
		} else {
			// If the record of this bucket id already exist, then read the record of the newest month
			// and use the free quota consumed of this record to init free quota item
			var newestTraffic BucketTrafficTable
			queryErr := s.db.Where("bucket_id = ?", bucketID).Order("month DESC").Limit(1).Find(&newestTraffic).Error
			if queryErr != nil {
				return queryErr
			}

			insertBucketTraffic = &BucketTrafficTable{
				BucketID:              bucketID,
				Month:                 yearMonth,
				FreeQuotaSize:         newestTraffic.FreeQuotaSize,
				FreeQuotaConsumedSize: newestTraffic.FreeQuotaConsumedSize,
				BucketName:            bucketName,
				ReadConsumedSize:      0,
				ChargedQuotaSize:      quota.ChargedQuotaSize,
				ModifiedTime:          time.Now(),
			}
		}

		result = tx.Create(insertBucketTraffic)
		if result.Error != nil && MysqlErrCode(result.Error) != ErrDuplicateEntryCode {
			return fmt.Errorf("failed to create bucket traffic table: %s", result.Error)
		}

		return nil
	})

	if err != nil {
		log.CtxErrorw(context.Background(), "init traffic table error ", "bucket name", bucketName, "error", err)
	}
	return err
}

// GetBucketTraffic return bucket traffic info by the year and month info
// year_month is the query bucket quota's month, like "2023-03"
func (s *SpDBImpl) GetBucketTraffic(bucketID uint64, yearMonth string) (traffic *corespdb.BucketTraffic, err error) {
	var (
		result      *gorm.DB
		queryReturn BucketTrafficTable
	)

	startTime := time.Now()
	defer func() {
		if err != nil {
			metrics.SPDBCounter.WithLabelValues(SPDBFailureGetBucketTraffic).Inc()
			metrics.SPDBTime.WithLabelValues(SPDBFailureGetBucketTraffic).Observe(
				time.Since(startTime).Seconds())
			return
		}
		metrics.SPDBCounter.WithLabelValues(SPDBSuccessGetBucketTraffic).Inc()
		metrics.SPDBTime.WithLabelValues(SPDBSuccessGetBucketTraffic).Observe(
			time.Since(startTime).Seconds())
	}()

	result = s.db.Where("bucket_id = ? and month = ?", bucketID, yearMonth).First(&queryReturn)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		err = result.Error
		return nil, err
	}
	if result.Error != nil {
		err = fmt.Errorf("failed to query bucket traffic table: %s", result.Error)
		return nil, err
	}
	return &corespdb.BucketTraffic{
		BucketID:              queryReturn.BucketID,
		YearMonth:             queryReturn.Month,
		FreeQuotaSize:         queryReturn.FreeQuotaSize,
		FreeQuotaConsumedSize: queryReturn.FreeQuotaConsumedSize,
		BucketName:            queryReturn.BucketName,
		ReadConsumedSize:      queryReturn.ReadConsumedSize,
		ChargedQuotaSize:      queryReturn.ChargedQuotaSize,
		ModifyTime:            queryReturn.ModifiedTime.Unix(),
	}, nil
}

// UpdateExtraQuota update the read consumed quota and free consumed quota in traffic db with the extra quota
func (s *SpDBImpl) UpdateExtraQuota(bucketID, extraQuota uint64) error {
	log.CtxErrorw(context.Background(), "begin to update extra quota for traffic db", "extra quota", extraQuota)
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var newestTraffic BucketTrafficTable
		var err error
		// lock the record of the newest month to update
		if err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("bucket_id = ? ", bucketID).Order("month DESC").Limit(1).Find(&newestTraffic).Error; err != nil {
			return fmt.Errorf("failed to query bucket traffic table: %v", err)
		}

		if newestTraffic.ReadConsumedSize < extraQuota {
			return fmt.Errorf("the extra quota %d to reimburse should be less than read consumed quota %d", extraQuota, newestTraffic.ReadConsumedSize)
		}
		updatedReadConsumed := newestTraffic.ReadConsumedSize - extraQuota

		// if the free quota has not exhaust even after consumed extra quota, the consumed free quota should be updated
		if newestTraffic.FreeQuotaSize-newestTraffic.FreeQuotaConsumedSize > 0 {
			if newestTraffic.FreeQuotaConsumedSize < extraQuota {
				return fmt.Errorf("the extra quota %d to reimburse should be less than read consumed quota %d", extraQuota, newestTraffic.ReadConsumedSize)
			}
			updatedFreeConsumed := newestTraffic.FreeQuotaConsumedSize - extraQuota

			err = tx.Model(&newestTraffic).
				Updates(BucketTrafficTable{
					ReadConsumedSize:      updatedReadConsumed,
					FreeQuotaConsumedSize: updatedFreeConsumed,
					ModifiedTime:          time.Now(),
				}).Error
		} else {
			// if the free quota has been exhausted, needed to compute the free quota consumed this month and compute if the extra data has contained free quota.
			// If the consumed quota minus the extra quota is less than the free quota remained this month, the consumed free quota should be updated.
			// for example, the freeQuota is 10G and remained 9G at the beginning of this month, and the consumedQuota is 10G after suffering 2G extra quota,
			// the consumedQuota should update to 8G and the remained freeQuota should be changed from 0G to 1G, consumed free quota change from 10G to 9G
			// if the freeQuota is 10G and remained 9G at the beginning of this month, but the consumedQuota is 13G after suffering 2G extra quota,
			// the consumedQuota should update to 11G and the consumed freeQuota should not be changed
			var secondaryNewestTraffic BucketTrafficTable
			var freeQuotaRemained uint64
			queryErr := tx.Where("bucket_id = ?", bucketID).Order("Month DESC").Offset(1).Limit(1).Find(&secondaryNewestTraffic).Error
			// the free quota remained at the beginning of this month should compute by the record of last month if it exists.
			// if not exists, the free quota remained at the beginning is free quota total size
			if queryErr != nil {
				if !errors.Is(queryErr, gorm.ErrRecordNotFound) {
					return queryErr
				} else {
					freeQuotaRemained = newestTraffic.FreeQuotaSize
				}
			} else {
				freeQuotaRemained = secondaryNewestTraffic.FreeQuotaSize - secondaryNewestTraffic.FreeQuotaConsumedSize
			}

			if updatedReadConsumed > freeQuotaRemained {
				// the extra data has not contained free quota, no need to update free consumed quota
				err = tx.Model(&newestTraffic).
					Updates(BucketTrafficTable{
						ReadConsumedSize: updatedReadConsumed,
						ModifiedTime:     time.Now(),
					}).Error
			} else {
				// the extra data has not contained free quota, no need to update free consumed quota
				exactRemainedFreeQuota := freeQuotaRemained - updatedReadConsumed
				exactConsumedFreeQuota := newestTraffic.FreeQuotaConsumedSize - exactRemainedFreeQuota
				err = tx.Model(&newestTraffic).
					Updates(BucketTrafficTable{
						ReadConsumedSize:      updatedReadConsumed,
						FreeQuotaConsumedSize: exactConsumedFreeQuota,
						ModifiedTime:          time.Now(),
					}).Error
			}

		}

		if err != nil {
			return fmt.Errorf("failed to update bucket traffic table: %v", err)
		}

		return nil
	})

	if err != nil {
		log.CtxErrorw(context.Background(), "fail to init fix extra quota ", "bucket id", bucketID, "error", err)
	}

	return err
}

// GetReadRecord return record list by time range
func (s *SpDBImpl) GetReadRecord(timeRange *corespdb.TrafficTimeRange) (records []*corespdb.ReadRecord, err error) {
	var (
		result       *gorm.DB
		queryReturns []ReadRecordTable
	)
	startTime := time.Now()
	defer func() {
		if err != nil {
			metrics.SPDBCounter.WithLabelValues(SPDBFailureGetReadRecord).Inc()
			metrics.SPDBTime.WithLabelValues(SPDBFailureGetReadRecord).Observe(
				time.Since(startTime).Seconds())
			return
		}
		metrics.SPDBCounter.WithLabelValues(SPDBSuccessGetReadRecord).Inc()
		metrics.SPDBTime.WithLabelValues(SPDBSuccessGetReadRecord).Observe(
			time.Since(startTime).Seconds())
	}()

	if timeRange.LimitNum <= 0 {
		result = s.db.Where("read_timestamp_us >= ? and read_timestamp_us < ?", timeRange.StartTimestampUs, timeRange.EndTimestampUs).
			Find(&queryReturns)
	} else {
		result = s.db.Where("read_timestamp_us >= ? and read_timestamp_us < ?", timeRange.StartTimestampUs, timeRange.EndTimestampUs).
			Limit(timeRange.LimitNum).Find(&queryReturns)
	}
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		err = result.Error
		return nil, err
	}
	if result.Error != nil {
		err = fmt.Errorf("failed to read record table: %s", result.Error)
		return records, err
	}
	for _, record := range queryReturns {
		records = append(records, &corespdb.ReadRecord{
			BucketID:        record.BucketID,
			ObjectID:        record.ObjectID,
			UserAddress:     record.UserAddress,
			BucketName:      record.BucketName,
			ObjectName:      record.ObjectName,
			ReadSize:        record.ReadSize,
			ReadTimestampUs: record.ReadTimestampUs,
		})
	}
	return records, nil
}

// GetBucketReadRecord return bucket record list by time range
func (s *SpDBImpl) GetBucketReadRecord(bucketID uint64, timeRange *corespdb.TrafficTimeRange) (records []*corespdb.ReadRecord, err error) {
	var (
		result       *gorm.DB
		queryReturns []ReadRecordTable
	)
	startTime := time.Now()
	defer func() {
		if err != nil {
			metrics.SPDBCounter.WithLabelValues(SPDBFailureGetBucketReadRecord).Inc()
			return
		}
		metrics.SPDBCounter.WithLabelValues(SPDBSuccessGetBucketReadRecord).Inc()
		metrics.SPDBTime.WithLabelValues(SPDBSuccessGetBucketReadRecord).Observe(
			time.Since(startTime).Seconds())
	}()

	if timeRange.LimitNum <= 0 {
		result = s.db.Where("read_timestamp_us >= ? and read_timestamp_us < ? and bucket_id = ?",
			timeRange.StartTimestampUs, timeRange.EndTimestampUs, bucketID).
			Find(&queryReturns)
	} else {
		result = s.db.Where("read_timestamp_us >= ? and read_timestamp_us < ? and bucket_id = ?",
			timeRange.StartTimestampUs, timeRange.EndTimestampUs, bucketID).
			Limit(timeRange.LimitNum).Find(&queryReturns)
	}
	if result.Error != nil {
		err = fmt.Errorf("failed to query read record table: %s", result.Error)
		return records, err
	}
	for _, record := range queryReturns {
		records = append(records, &corespdb.ReadRecord{
			BucketID:        record.BucketID,
			ObjectID:        record.ObjectID,
			UserAddress:     record.UserAddress,
			BucketName:      record.BucketName,
			ObjectName:      record.ObjectName,
			ReadSize:        record.ReadSize,
			ReadTimestampUs: record.ReadTimestampUs,
		})
	}
	return records, nil
}

// GetObjectReadRecord return object record list by time range
func (s *SpDBImpl) GetObjectReadRecord(objectID uint64, timeRange *corespdb.TrafficTimeRange) (records []*corespdb.ReadRecord, err error) {
	var (
		result       *gorm.DB
		queryReturns []ReadRecordTable
	)
	startTime := time.Now()
	defer func() {
		if err != nil {
			metrics.SPDBCounter.WithLabelValues(SPDBFailureGetObjectReadRecord).Inc()
			return
		}
		metrics.SPDBCounter.WithLabelValues(SPDBSuccessGetObjectReadRecord).Inc()
		metrics.SPDBTime.WithLabelValues(SPDBSuccessGetObjectReadRecord).Observe(
			time.Since(startTime).Seconds())
	}()

	if timeRange.LimitNum <= 0 {
		result = s.db.Where("read_timestamp_us >= ? and read_timestamp_us < ? and object_id = ?",
			timeRange.StartTimestampUs, timeRange.EndTimestampUs, objectID).
			Find(&queryReturns)
	} else {
		result = s.db.Where("read_timestamp_us >= ? and read_timestamp_us < ? and object_id = ?",
			timeRange.StartTimestampUs, timeRange.EndTimestampUs, objectID).
			Limit(timeRange.LimitNum).Find(&queryReturns)
	}
	if result.Error != nil {
		err = fmt.Errorf("failed to query read record table: %s", result.Error)
		return records, err
	}
	for _, record := range queryReturns {
		records = append(records, &corespdb.ReadRecord{
			BucketID:        record.BucketID,
			ObjectID:        record.ObjectID,
			UserAddress:     record.UserAddress,
			BucketName:      record.BucketName,
			ObjectName:      record.ObjectName,
			ReadSize:        record.ReadSize,
			ReadTimestampUs: record.ReadTimestampUs,
		})
	}
	return records, nil
}

// GetUserReadRecord return user record list by time range
func (s *SpDBImpl) GetUserReadRecord(userAddress string, timeRange *corespdb.TrafficTimeRange) (records []*corespdb.ReadRecord, err error) {
	var (
		result       *gorm.DB
		queryReturns []ReadRecordTable
	)
	startTime := time.Now()
	defer func() {
		if err != nil {
			metrics.SPDBCounter.WithLabelValues(SPDBFailureGetUserReadRecord).Inc()
			return
		}
		metrics.SPDBCounter.WithLabelValues(SPDBSuccessGetUserReadRecord).Inc()
		metrics.SPDBTime.WithLabelValues(SPDBSuccessGetUserReadRecord).Observe(
			time.Since(startTime).Seconds())
	}()

	if timeRange.LimitNum <= 0 {
		result = s.db.Where("read_timestamp_us >= ? and read_timestamp_us < ? and user_address = ?",
			timeRange.StartTimestampUs, timeRange.EndTimestampUs, userAddress).
			Find(&queryReturns)
	} else {
		result = s.db.Where("read_timestamp_us >= ? and read_timestamp_us < ? and user_address = ?",
			timeRange.StartTimestampUs, timeRange.EndTimestampUs, userAddress).
			Limit(timeRange.LimitNum).Find(&queryReturns)
	}
	if result.Error != nil {
		err = fmt.Errorf("failed to query read record table: %s", result.Error)
		return records, err
	}
	for _, record := range queryReturns {
		records = append(records, &corespdb.ReadRecord{
			BucketID:        record.BucketID,
			ObjectID:        record.ObjectID,
			UserAddress:     record.UserAddress,
			BucketName:      record.BucketName,
			ObjectName:      record.ObjectName,
			ReadSize:        record.ReadSize,
			ReadTimestampUs: record.ReadTimestampUs,
		})
	}
	return records, nil
}
