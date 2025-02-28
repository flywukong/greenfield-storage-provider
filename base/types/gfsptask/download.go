package gfsptask

import (
	"fmt"
	"time"

	"github.com/bnb-chain/greenfield-storage-provider/base/types/gfsplimit"
	corercmgr "github.com/bnb-chain/greenfield-storage-provider/core/rcmgr"
	coretask "github.com/bnb-chain/greenfield-storage-provider/core/task"
	storagetypes "github.com/bnb-chain/greenfield/x/storage/types"
)

var _ coretask.DownloadObjectTask = &GfSpDownloadObjectTask{}
var _ coretask.DownloadPieceTask = &GfSpDownloadPieceTask{}
var _ coretask.ChallengePieceTask = &GfSpChallengePieceTask{}

func (m *GfSpDownloadObjectTask) InitDownloadObjectTask(object *storagetypes.ObjectInfo, bucket *storagetypes.BucketInfo,
	params *storagetypes.Params, priority coretask.TPriority, userAddress string, low int64, high int64, timeout int64, maxRetry int64) {
	m.Reset()
	m.Task = &GfSpTask{}
	m.SetCreateTime(time.Now().Unix())
	m.SetUpdateTime(time.Now().Unix())
	m.SetObjectInfo(object)
	m.SetBucketInfo(bucket)
	m.SetStorageParams(params)
	m.SetUserAddress(userAddress)
	m.Low = low
	m.High = high
	m.SetPriority(priority)
	m.SetTimeout(timeout)
	m.SetMaxRetry(maxRetry)
}

func (m *GfSpDownloadObjectTask) Key() coretask.TKey {
	return GfSpDownloadObjectTaskKey(
		m.GetObjectInfo().GetBucketName(),
		m.GetObjectInfo().GetObjectName(),
		m.GetObjectInfo().Id.String(),
		m.GetLow(),
		m.GetHigh())
}

func (m *GfSpDownloadObjectTask) Type() coretask.TType {
	return coretask.TypeTaskDownloadObject
}

func (m *GfSpDownloadObjectTask) Info() string {
	return fmt.Sprintf("key[%s], type[%s], priority[%d], limit[%s], %s",
		m.Key(), coretask.TaskTypeName(m.Type()), m.GetPriority(),
		m.EstimateLimit().String(), m.GetTask().Info())
}

func (m *GfSpDownloadObjectTask) GetAddress() string {
	return m.GetTask().GetAddress()
}

func (m *GfSpDownloadObjectTask) SetAddress(address string) {
	m.GetTask().SetAddress(address)
}

func (m *GfSpDownloadObjectTask) GetCreateTime() int64 {
	return m.GetTask().GetCreateTime()
}

func (m *GfSpDownloadObjectTask) SetCreateTime(time int64) {
	m.GetTask().SetCreateTime(time)
}

func (m *GfSpDownloadObjectTask) GetUpdateTime() int64 {
	return m.GetTask().GetUpdateTime()
}

func (m *GfSpDownloadObjectTask) SetUpdateTime(time int64) {
	m.GetTask().SetUpdateTime(time)
}

func (m *GfSpDownloadObjectTask) GetTimeout() int64 {
	return m.GetTask().GetTimeout()
}

func (m *GfSpDownloadObjectTask) SetTimeout(time int64) {
	m.GetTask().SetTimeout(time)
}

func (m *GfSpDownloadObjectTask) ExceedTimeout() bool {
	return m.GetTask().ExceedTimeout()
}

func (m *GfSpDownloadObjectTask) GetRetry() int64 {
	return m.GetTask().GetRetry()
}

func (m *GfSpDownloadObjectTask) IncRetry() {
	m.GetTask().IncRetry()
}

func (m *GfSpDownloadObjectTask) SetRetry(retry int) {
	m.GetTask().SetRetry(retry)
}

func (m *GfSpDownloadObjectTask) GetMaxRetry() int64 {
	return m.GetTask().GetMaxRetry()
}

func (m *GfSpDownloadObjectTask) SetMaxRetry(limit int64) {
	m.GetTask().SetMaxRetry(limit)
}

func (m *GfSpDownloadObjectTask) ExceedRetry() bool {
	return m.GetTask().ExceedRetry()
}

func (m *GfSpDownloadObjectTask) Expired() bool {
	return m.GetTask().Expired()
}

func (m *GfSpDownloadObjectTask) GetPriority() coretask.TPriority {
	return m.GetTask().GetPriority()
}

func (m *GfSpDownloadObjectTask) SetPriority(priority coretask.TPriority) {
	m.GetTask().SetPriority(priority)
}

func (m *GfSpDownloadObjectTask) EstimateLimit() corercmgr.Limit {
	l := &gfsplimit.GfSpLimit{Memory: m.GetSize()}
	l.Add(LimitEstimateByPriority(m.GetPriority()))
	return l
}

func (m *GfSpDownloadObjectTask) GetUserAddress() string {
	return m.GetTask().GetUserAddress()
}

func (m *GfSpDownloadObjectTask) SetUserAddress(address string) {
	m.GetTask().SetUserAddress(address)
}

func (m *GfSpDownloadObjectTask) SetLogs(logs string) {
	m.GetTask().SetLogs(logs)
}

func (m *GfSpDownloadObjectTask) GetLogs() string {
	return m.GetTask().GetLogs()
}

func (m *GfSpDownloadObjectTask) AppendLog(log string) {
	m.GetTask().AppendLog(log)
}

func (m *GfSpDownloadObjectTask) Error() error {
	return m.GetTask().Error()
}

func (m *GfSpDownloadObjectTask) SetError(err error) {
	m.GetTask().SetError(err)
}

func (m *GfSpDownloadObjectTask) GetSize() int64 {
	if m.GetHigh() < m.GetLow() {
		return 0
	}
	return m.GetHigh() - m.GetLow() + 1
}

func (m *GfSpDownloadObjectTask) SetObjectInfo(object *storagetypes.ObjectInfo) {
	m.ObjectInfo = object
}

func (m *GfSpDownloadObjectTask) SetStorageParams(params *storagetypes.Params) {
	m.StorageParams = params
}

func (m *GfSpDownloadObjectTask) SetBucketInfo(bucket *storagetypes.BucketInfo) {
	m.BucketInfo = bucket
}

func (m *GfSpDownloadPieceTask) InitDownloadPieceTask(object *storagetypes.ObjectInfo, bucket *storagetypes.BucketInfo,
	params *storagetypes.Params, priority coretask.TPriority, enableCheck bool, userAddress string, totalSize uint64,
	pieceKey string, pieceOffset uint64, pieceLength uint64, timeout int64, maxRetry int64) {
	m.Reset()
	m.Task = &GfSpTask{}
	m.SetCreateTime(time.Now().Unix())
	m.SetUpdateTime(time.Now().Unix())
	m.SetObjectInfo(object)
	m.SetBucketInfo(bucket)
	m.SetStorageParams(params)
	m.SetUserAddress(userAddress)
	m.SetPriority(priority)
	m.SetTimeout(timeout)
	m.SetMaxRetry(maxRetry)
	m.EnableCheck = enableCheck
	m.TotalSize = totalSize
	m.PieceKey = pieceKey
	m.PieceOffset = pieceOffset
	m.PieceLength = pieceLength
}

func (m *GfSpDownloadPieceTask) Key() coretask.TKey {
	return GfSpDownloadPieceTaskKey(
		m.GetObjectInfo().GetBucketName(),
		m.GetObjectInfo().GetObjectName(),
		m.GetPieceKey(),
		m.GetPieceOffset(),
		m.GetPieceLength())
}

func (m *GfSpDownloadPieceTask) Type() coretask.TType {
	return coretask.TypeTaskDownloadPiece
}

func (m *GfSpDownloadPieceTask) Info() string {
	return fmt.Sprintf("key[%s], type[%s], priority[%d], limit[%s], %s",
		m.Key(), coretask.TaskTypeName(m.Type()), m.GetPriority(),
		m.EstimateLimit().String(), m.GetTask().Info())
}

func (m *GfSpDownloadPieceTask) GetAddress() string {
	return m.GetTask().GetAddress()
}

func (m *GfSpDownloadPieceTask) SetAddress(address string) {
	m.GetTask().SetAddress(address)
}

func (m *GfSpDownloadPieceTask) GetCreateTime() int64 {
	return m.GetTask().GetCreateTime()
}

func (m *GfSpDownloadPieceTask) SetCreateTime(time int64) {
	m.GetTask().SetCreateTime(time)
}

func (m *GfSpDownloadPieceTask) GetUpdateTime() int64 {
	return m.GetTask().GetUpdateTime()
}

func (m *GfSpDownloadPieceTask) SetUpdateTime(time int64) {
	m.GetTask().SetUpdateTime(time)
}

func (m *GfSpDownloadPieceTask) GetTimeout() int64 {
	return m.GetTask().GetTimeout()
}

func (m *GfSpDownloadPieceTask) SetTimeout(time int64) {
	m.GetTask().SetTimeout(time)
}

func (m *GfSpDownloadPieceTask) ExceedTimeout() bool {
	return m.GetTask().ExceedTimeout()
}

func (m *GfSpDownloadPieceTask) GetRetry() int64 {
	return m.GetTask().GetRetry()
}

func (m *GfSpDownloadPieceTask) IncRetry() {
	m.GetTask().IncRetry()
}

func (m *GfSpDownloadPieceTask) SetRetry(retry int) {
	m.GetTask().SetRetry(retry)
}

func (m *GfSpDownloadPieceTask) GetMaxRetry() int64 {
	return m.GetTask().GetMaxRetry()
}

func (m *GfSpDownloadPieceTask) SetMaxRetry(limit int64) {
	m.GetTask().SetMaxRetry(limit)
}

func (m *GfSpDownloadPieceTask) ExceedRetry() bool {
	return m.GetTask().ExceedRetry()
}

func (m *GfSpDownloadPieceTask) Expired() bool {
	return m.GetTask().Expired()
}

func (m *GfSpDownloadPieceTask) GetPriority() coretask.TPriority {
	return m.GetTask().GetPriority()
}

func (m *GfSpDownloadPieceTask) SetPriority(priority coretask.TPriority) {
	m.GetTask().SetPriority(priority)
}

func (m *GfSpDownloadPieceTask) EstimateLimit() corercmgr.Limit {
	l := &gfsplimit.GfSpLimit{Memory: m.GetSize()}
	l.Add(LimitEstimateByPriority(m.GetPriority()))
	return l
}

func (m *GfSpDownloadPieceTask) SetLogs(logs string) {
	m.GetTask().SetLogs(logs)
}

func (m *GfSpDownloadPieceTask) GetLogs() string {
	return m.GetTask().GetLogs()
}

func (m *GfSpDownloadPieceTask) AppendLog(log string) {
	m.GetTask().AppendLog(log)
}

func (m *GfSpDownloadPieceTask) Error() error {
	return m.GetTask().Error()
}

func (m *GfSpDownloadPieceTask) SetError(err error) {
	m.GetTask().SetError(err)
}

func (m *GfSpDownloadPieceTask) GetSize() int64 {
	return int64(m.GetPieceLength())
}

func (m *GfSpDownloadPieceTask) SetObjectInfo(object *storagetypes.ObjectInfo) {
	m.ObjectInfo = object
}

func (m *GfSpDownloadPieceTask) GetUserAddress() string {
	return m.GetTask().GetUserAddress()
}

func (m *GfSpDownloadPieceTask) SetUserAddress(address string) {
	m.GetTask().SetUserAddress(address)
}

func (m *GfSpDownloadPieceTask) SetStorageParams(params *storagetypes.Params) {
	m.StorageParams = params
}

func (m *GfSpDownloadPieceTask) SetBucketInfo(bucket *storagetypes.BucketInfo) {
	m.BucketInfo = bucket
}

func (m *GfSpChallengePieceTask) InitChallengePieceTask(object *storagetypes.ObjectInfo, bucket *storagetypes.BucketInfo,
	params *storagetypes.Params, priority coretask.TPriority, userAddress string, replicateIdx int32, segmentIdx uint32,
	timeout int64, retry int64) {
	m.Reset()
	m.Task = &GfSpTask{}
	m.SetCreateTime(time.Now().Unix())
	m.SetUpdateTime(time.Now().Unix())
	m.SetBucketInfo(bucket)
	m.SetObjectInfo(object)
	m.SetStorageParams(params)
	m.SetUserAddress(userAddress)
	m.SetRedundancyIdx(replicateIdx)
	m.SetSegmentIdx(segmentIdx)
	m.SetPriority(priority)
	m.SetTimeout(timeout)
	m.SetMaxRetry(retry)
}

func (m *GfSpChallengePieceTask) Key() coretask.TKey {
	return GfSpChallengePieceTaskKey(
		m.GetObjectInfo().GetBucketName(),
		m.GetObjectInfo().GetObjectName(),
		m.GetObjectInfo().Id.String(),
		m.GetSegmentIdx(),
		m.GetRedundancyIdx(),
		m.GetUserAddress())
}

func (m *GfSpChallengePieceTask) Type() coretask.TType {
	return coretask.TypeTaskChallengePiece
}

func (m *GfSpChallengePieceTask) Info() string {
	return fmt.Sprintf("key[%s], type[%s], priority[%d], limit[%s], rIdx[%d], sIdx[%d], %s",
		m.Key(), coretask.TaskTypeName(m.Type()), m.GetPriority(), m.EstimateLimit().String(),
		m.GetRedundancyIdx(), m.GetSegmentIdx(), m.GetTask().Info())
}

func (m *GfSpChallengePieceTask) GetAddress() string {
	return m.GetTask().GetAddress()
}

func (m *GfSpChallengePieceTask) SetAddress(address string) {
	m.GetTask().SetAddress(address)
}

func (m *GfSpChallengePieceTask) GetCreateTime() int64 {
	return m.GetTask().GetCreateTime()
}

func (m *GfSpChallengePieceTask) SetCreateTime(time int64) {
	m.GetTask().SetCreateTime(time)
}

func (m *GfSpChallengePieceTask) GetUpdateTime() int64 {
	return m.GetTask().GetUpdateTime()
}

func (m *GfSpChallengePieceTask) SetUpdateTime(time int64) {
	m.GetTask().SetUpdateTime(time)
}

func (m *GfSpChallengePieceTask) GetTimeout() int64 {
	return m.GetTask().GetTimeout()
}

func (m *GfSpChallengePieceTask) SetTimeout(time int64) {
	m.GetTask().SetTimeout(time)
}

func (m *GfSpChallengePieceTask) ExceedTimeout() bool {
	return m.GetTask().ExceedTimeout()
}

func (m *GfSpChallengePieceTask) GetRetry() int64 {
	return m.GetTask().GetRetry()
}

func (m *GfSpChallengePieceTask) IncRetry() {
	m.GetTask().IncRetry()
}

func (m *GfSpChallengePieceTask) SetRetry(retry int) {
	m.GetTask().SetRetry(retry)
}

func (m *GfSpChallengePieceTask) GetMaxRetry() int64 {
	return m.GetTask().GetMaxRetry()
}

func (m *GfSpChallengePieceTask) SetMaxRetry(limit int64) {
	m.GetTask().SetMaxRetry(limit)
}

func (m *GfSpChallengePieceTask) ExceedRetry() bool {
	return m.GetTask().ExceedRetry()
}

func (m *GfSpChallengePieceTask) Expired() bool {
	return m.GetTask().Expired()
}

func (m *GfSpChallengePieceTask) GetPriority() coretask.TPriority {
	return m.GetTask().GetPriority()
}

func (m *GfSpChallengePieceTask) SetPriority(priority coretask.TPriority) {
	m.GetTask().SetPriority(priority)
}

func (m *GfSpChallengePieceTask) EstimateLimit() corercmgr.Limit {
	var memSize int64
	payloadSize := m.GetObjectInfo().GetPayloadSize()
	maxSegmentSize := m.GetStorageParams().VersionedParams.GetMaxSegmentSize()
	dataChunkNum := m.GetStorageParams().VersionedParams.GetRedundantDataChunkNum()
	segmentCount := m.GetObjectInfo().GetPayloadSize() / m.GetStorageParams().VersionedParams.GetMaxSegmentSize()
	if m.GetObjectInfo().GetPayloadSize()%m.GetStorageParams().VersionedParams.GetMaxSegmentSize() != 0 {
		segmentCount++
	}
	if m.GetRedundancyIdx() < 0 {
		if segmentCount == 1 {
			memSize = int64(payloadSize)
		} else if m.GetSegmentIdx() == uint32(segmentCount-1) {
			memSize = int64(payloadSize) - (int64(segmentCount)-1)*int64(maxSegmentSize)
		} else {
			memSize = int64(maxSegmentSize)
		}
	} else {
		if segmentCount == 1 {
			memSize = int64(payloadSize) / int64(dataChunkNum)
		} else if m.GetSegmentIdx() == uint32(segmentCount-1) {
			memSize = int64(payloadSize) - (int64(segmentCount)-1)*int64(maxSegmentSize)/int64(dataChunkNum)
		} else {
			memSize = int64(maxSegmentSize) / int64(dataChunkNum)
		}
	}
	l := &gfsplimit.GfSpLimit{Memory: memSize}
	l.Add(LimitEstimateByPriority(m.GetPriority()))
	return l
}

func (m *GfSpChallengePieceTask) SetLogs(logs string) {
	m.GetTask().SetLogs(logs)
}

func (m *GfSpChallengePieceTask) GetLogs() string {
	return m.GetTask().GetLogs()
}

func (m *GfSpChallengePieceTask) AppendLog(log string) {
	m.GetTask().AppendLog(log)
}

func (m *GfSpChallengePieceTask) Error() error {
	return m.GetTask().Error()
}

func (m *GfSpChallengePieceTask) SetError(err error) {
	m.GetTask().SetError(err)
}

func (m *GfSpChallengePieceTask) SetObjectInfo(object *storagetypes.ObjectInfo) {
	m.ObjectInfo = object
}

func (m *GfSpChallengePieceTask) SetStorageParams(params *storagetypes.Params) {
	m.StorageParams = params
}

func (m *GfSpChallengePieceTask) SetBucketInfo(bucket *storagetypes.BucketInfo) {
	m.BucketInfo = bucket
}

func (m *GfSpChallengePieceTask) GetUserAddress() string {
	return m.GetTask().GetUserAddress()
}

func (m *GfSpChallengePieceTask) SetUserAddress(address string) {
	m.GetTask().SetUserAddress(address)
}

func (m *GfSpChallengePieceTask) SetSegmentIdx(idx uint32) {
	m.SegmentIdx = idx
}

func (m *GfSpChallengePieceTask) SetRedundancyIdx(idx int32) {
	m.RedundancyIdx = idx
}

func (m *GfSpChallengePieceTask) SetIntegrityHash(checksum []byte) {
	m.IntegrityHash = checksum
}

func (m *GfSpChallengePieceTask) SetPieceHash(checksums [][]byte) {
	m.PieceHash = checksums
}

func (m *GfSpChallengePieceTask) SetPieceDataSize(size int64) {
	m.PieceDataSize = size
}
