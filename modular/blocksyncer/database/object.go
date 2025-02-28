package database

import (
	"context"

	"github.com/forbole/juno/v4/common"
	"github.com/forbole/juno/v4/models"
	"gorm.io/gorm/clause"

	"github.com/bnb-chain/greenfield-storage-provider/store/bsdb"
)

func (db *DB) SaveObject(ctx context.Context, object *models.Object) error {
	err := db.Db.WithContext(ctx).Table(bsdb.GetObjectsTableName(object.BucketName)).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "object_id"}},
		UpdateAll: true,
	}).Create(object).Error
	return err
}

func (db *DB) BatchSaveObject(ctx context.Context, objects map[string][]*models.Object) error {
	for k, v := range objects {
		err := db.Db.WithContext(ctx).Table(bsdb.GetObjectsTableName(k)).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "object_id"}},
			UpdateAll: true,
		}).Create(v).Error
		if err != nil {
			return err
		}
	}
	return nil
}

func (db *DB) GetObjectList(ctx context.Context, objects map[string][]common.Hash) ([]*models.Object, error) {
	res := make([]*models.Object, 0)
	for k, v := range objects {
		var tmp []*models.Object
		err := db.Db.WithContext(ctx).Table(bsdb.GetObjectsTableName(k)).Where("object_id IN ? AND removed IS NOT TRUE", v).Find(&tmp).Error
		if err != nil {
			return nil, err
		}
		res = append(res, tmp...)
	}
	return res, nil
}

func (db *DB) UpdateObject(ctx context.Context, object *models.Object) error {
	err := db.Db.WithContext(ctx).Table(bsdb.GetObjectsTableName(object.BucketName)).Where("object_id = ?", object.ObjectID).Updates(object).Error
	return err
}

func (db *DB) GetObject(ctx context.Context, objectId common.Hash) (*models.Object, error) {
	var object models.Object
	bucketName, err := db.GetBucketNameByObjectID(objectId)

	if err != nil {
		return nil, err
	}

	err = db.Db.WithContext(ctx).Table(bsdb.GetObjectsTableName(bucketName)).Where(
		"object_id = ? AND removed IS NOT TRUE", objectId).Find(&object).Error
	if err != nil {
		return nil, err
	}
	return &object, nil
}

// GetBucketNameByObjectID get bucket name info by an object id
func (b *DB) GetBucketNameByObjectID(objectID common.Hash) (string, error) {
	var (
		objectIdMap *bsdb.ObjectIDMap
		err         error
	)

	err = b.Db.Table((&bsdb.ObjectIDMap{}).TableName()).
		Select("*").
		Where("object_id = ?", objectID).
		Take(&objectIdMap).Error

	return objectIdMap.BucketName, err
}
