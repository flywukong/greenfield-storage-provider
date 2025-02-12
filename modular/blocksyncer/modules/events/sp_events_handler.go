package events

import (
	"context"
	"errors"

	abci "github.com/cometbft/cometbft/abci/types"
	tmctypes "github.com/cometbft/cometbft/rpc/core/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/forbole/juno/v4/common"
	"github.com/forbole/juno/v4/log"

	"github.com/bnb-chain/greenfield-storage-provider/store/bsdb"
	storagetypes "github.com/bnb-chain/greenfield/x/storage/types"
	vgtypes "github.com/bnb-chain/greenfield/x/virtualgroup/types"
)

var (
	EventMigrationBucket         = proto.MessageName(&storagetypes.EventMigrationBucket{})
	EventCompleteMigrationBucket = proto.MessageName(&storagetypes.EventCompleteMigrationBucket{})
	EventCancelMigrationBucket   = proto.MessageName(&storagetypes.EventCancelMigrationBucket{})

	EventSwapOut         = proto.MessageName(&vgtypes.EventSwapOut{})
	EventCompleteSwapOut = proto.MessageName(&vgtypes.EventCompleteSwapOut{})
	EventCancelSwapOut   = proto.MessageName(&vgtypes.EventCancelSwapOut{})

	EventSpExit         = proto.MessageName(&vgtypes.EventStorageProviderExit{})
	EventCompleteSpExit = proto.MessageName(&vgtypes.EventCompleteStorageProviderExit{})
)

var SpExitEvents = map[string]bool{
	EventMigrationBucket:         true,
	EventCompleteMigrationBucket: true,
	EventCancelMigrationBucket:   true,
	EventSwapOut:                 true,
	EventCompleteSwapOut:         true,
	EventCancelSwapOut:           true,
	EventSpExit:                  true,
	EventCompleteSpExit:          true,
}

func (m *Module) ExtractEvent(ctx context.Context, block *tmctypes.ResultBlock, txHash common.Hash, event sdk.Event) (interface{}, error) {
	return nil, nil
}

func (m *Module) HandleEvent(ctx context.Context, block *tmctypes.ResultBlock, txHash common.Hash, event sdk.Event) error {
	if !SpExitEvents[event.Type] {
		return nil
	}

	typedEvent, err := sdk.ParseTypedEvent(abci.Event(event))
	if err != nil {
		log.Errorw("parse typed events error", "module", m.Name(), "event", event, "err", err)
		return err
	}

	log.Debugw("HandleEvent", "event", event)
	switch event.Type {
	case EventMigrationBucket:
		migrationBucket, ok := typedEvent.(*storagetypes.EventMigrationBucket)
		if !ok {
			log.Errorw("type assert error", "type", "EventMigrationBucket", "event", typedEvent)
			return errors.New("migration bucket event assert error")
		}
		log.Debugw("HandleEvent EventMigrationBucket", "migrationBucket", migrationBucket)
		return m.handleMigrationBucket(ctx, block, txHash, migrationBucket)
	case EventSwapOut:
		swapOut, ok := typedEvent.(*vgtypes.EventSwapOut)
		if !ok {
			log.Errorw("type assert error", "type", "EventSwapOut", "event", typedEvent)
			return errors.New("swap out event assert error")
		}
		return m.handleSwapOut(ctx, block, txHash, swapOut)
	case EventCompleteMigrationBucket:
		migrationBucket, ok := typedEvent.(*storagetypes.EventCompleteMigrationBucket)
		if !ok {
			log.Errorw("type assert error", "type", "EventCompleteMigrationBucket", "event", typedEvent)
			return errors.New("migration bucket complete event assert error")
		}
		return m.handleCompleteMigrationBucket(ctx, block, txHash, migrationBucket)
	case EventSpExit:
		storageProviderExit, ok := typedEvent.(*vgtypes.EventStorageProviderExit)
		if !ok {
			log.Errorw("type assert error", "type", "EventStorageProviderExit", "event", typedEvent)
			return errors.New("storage provider Exit event assert error")
		}
		return m.handleStorageProviderExit(ctx, block, txHash, storageProviderExit)
	case EventCompleteSpExit:
		completeStorageProviderExit, ok := typedEvent.(*vgtypes.EventCompleteStorageProviderExit)
		if !ok {
			log.Errorw("type assert error", "type", "EventCompleteStorageProviderExit", "event", typedEvent)
			return errors.New("complete storage provider Exit event assert error")
		}
		return m.handleCompleteStorageProviderExit(ctx, block, txHash, completeStorageProviderExit)
	case EventCancelMigrationBucket:
		cancelMigrationBucket, ok := typedEvent.(*storagetypes.EventCancelMigrationBucket)
		if !ok {
			log.Errorw("type assert error", "type", "EventCancelMigrationBucket", "event", typedEvent)
			return errors.New("cancel Migration Bucket event assert error")
		}
		return m.handleCancelMigrationBucket(ctx, block, txHash, cancelMigrationBucket)
	case EventCancelSwapOut:
		cancelSwapOut, ok := typedEvent.(*vgtypes.EventCancelSwapOut)
		if !ok {
			log.Errorw("type assert error", "type", "EventCancelSwapOut", "event", typedEvent)
			return errors.New("cancel swap out event assert error")
		}
		return m.handleCancelSwapOut(ctx, block, txHash, cancelSwapOut)
	case EventCompleteSwapOut:
		completeSwapOut, ok := typedEvent.(*vgtypes.EventCompleteSwapOut)
		if !ok {
			log.Errorw("type assert error", "type", "EventCompleteSwapOut", "event", typedEvent)
			return errors.New("complete swap out event assert error")
		}
		return m.handleCompleteSwapOut(ctx, block, txHash, completeSwapOut)
	default:
		return nil
	}
}

func (m *Module) handleMigrationBucket(ctx context.Context, block *tmctypes.ResultBlock, txHash common.Hash, migrationBucket *storagetypes.EventMigrationBucket) error {
	eventMigrateBucketItem := &bsdb.EventMigrationBucket{
		BucketID:       common.BigToHash(migrationBucket.BucketId.BigInt()),
		Operator:       common.HexToAddress(migrationBucket.Operator),
		BucketName:     migrationBucket.BucketName,
		DstPrimarySpId: migrationBucket.DstPrimarySpId,

		CreateAt:     block.Block.Height,
		CreateTxHash: txHash,
		CreateTime:   block.Block.Time.UTC().Unix(),
	}

	return m.db.SaveEventMigrationBucket(ctx, eventMigrateBucketItem)

}

func (m *Module) handleCompleteMigrationBucket(ctx context.Context, block *tmctypes.ResultBlock, txHash common.Hash, completeMigrationBucket *storagetypes.EventCompleteMigrationBucket) error {

	eventCompleteMigrateBucketItem := &bsdb.EventCompleteMigrationBucket{
		BucketID:                   common.BigToHash(completeMigrationBucket.BucketId.BigInt()),
		Operator:                   common.HexToAddress(completeMigrationBucket.Operator),
		BucketName:                 completeMigrationBucket.BucketName,
		GlobalVirtualGroupFamilyId: completeMigrationBucket.GlobalVirtualGroupFamilyId,
		SrcPrimarySpId:             completeMigrationBucket.SrcPrimarySpId,

		CreateAt:     block.Block.Height,
		CreateTxHash: txHash,
		CreateTime:   block.Block.Time.UTC().Unix(),
	}

	return m.db.SaveEventCompleteMigrationBucket(ctx, eventCompleteMigrateBucketItem)
}

func (m *Module) handleSwapOut(ctx context.Context, block *tmctypes.ResultBlock, txHash common.Hash, swapOut *vgtypes.EventSwapOut) error {

	eventSwapOut := &bsdb.EventSwapOut{
		StorageProviderId:          swapOut.StorageProviderId,
		GlobalVirtualGroupFamilyId: swapOut.GlobalVirtualGroupFamilyId,
		GlobalVirtualGroupIds:      swapOut.GlobalVirtualGroupIds,
		SuccessorSpId:              swapOut.SuccessorSpId,

		CreateAt:     block.Block.Height,
		CreateTxHash: txHash,
		CreateTime:   block.Block.Time.UTC().Unix(),
	}

	return m.db.SaveEventSwapOut(ctx, eventSwapOut)
}

func (m *Module) handleCancelSwapOut(ctx context.Context, block *tmctypes.ResultBlock, txHash common.Hash, cancelSwapOut *vgtypes.EventCancelSwapOut) error {

	eventCancelSwapOut := &bsdb.EventCancelSwapOut{
		StorageProviderId:          cancelSwapOut.StorageProviderId,
		GlobalVirtualGroupFamilyId: cancelSwapOut.GlobalVirtualGroupFamilyId,
		GlobalVirtualGroupIds:      cancelSwapOut.GlobalVirtualGroupIds,
		SuccessorSpId:              cancelSwapOut.SuccessorSpId,

		CreateAt:     block.Block.Height,
		CreateTxHash: txHash,
		CreateTime:   block.Block.Time.UTC().Unix(),
	}

	return m.db.SaveEventCancelSwapOut(ctx, eventCancelSwapOut)
}

func (m *Module) handleCompleteSwapOut(ctx context.Context, block *tmctypes.ResultBlock, txHash common.Hash, completeSwapOut *vgtypes.EventCompleteSwapOut) error {

	eventCompleteSwapOut := &bsdb.EventCompleteSwapOut{
		StorageProviderId:          completeSwapOut.StorageProviderId,
		SrcStorageProviderId:       completeSwapOut.SrcStorageProviderId,
		GlobalVirtualGroupFamilyId: completeSwapOut.GlobalVirtualGroupFamilyId,
		GlobalVirtualGroupIds:      completeSwapOut.GlobalVirtualGroupIds,

		CreateAt:     block.Block.Height,
		CreateTxHash: txHash,
		CreateTime:   block.Block.Time.UTC().Unix(),
	}

	return m.db.SaveEventCompleteSwapOut(ctx, eventCompleteSwapOut)
}

func (m *Module) handleStorageProviderExit(ctx context.Context, block *tmctypes.ResultBlock, txHash common.Hash, storageProviderExit *vgtypes.EventStorageProviderExit) error {

	eventSPExit := &bsdb.EventStorageProviderExit{
		StorageProviderId: storageProviderExit.StorageProviderId,
		OperatorAddress:   common.HexToAddress(storageProviderExit.OperatorAddress),

		CreateAt:     block.Block.Height,
		CreateTxHash: txHash,
		CreateTime:   block.Block.Time.UTC().Unix(),
	}

	return m.db.SaveEventSPExit(ctx, eventSPExit)
}

func (m *Module) handleCompleteStorageProviderExit(ctx context.Context, block *tmctypes.ResultBlock, txHash common.Hash, completeStorageProviderExit *vgtypes.EventCompleteStorageProviderExit) error {

	eventSPCompleteExit := &bsdb.EventCompleteStorageProviderExit{
		StorageProviderId: completeStorageProviderExit.StorageProviderId,
		OperatorAddress:   common.HexToAddress(completeStorageProviderExit.OperatorAddress),
		TotalDeposit:      (*common.Big)(completeStorageProviderExit.TotalDeposit.BigInt()),

		CreateAt:     block.Block.Height,
		CreateTxHash: txHash,
		CreateTime:   block.Block.Time.UTC().Unix(),
	}

	return m.db.SaveEventSPCompleteExit(ctx, eventSPCompleteExit)
}

func (m *Module) handleCancelMigrationBucket(ctx context.Context, block *tmctypes.ResultBlock, txHash common.Hash, cancelMigrationBucket *storagetypes.EventCancelMigrationBucket) error {

	eventCancelMigrationBucket := &bsdb.EventCancelMigrationBucket{
		BucketID:   common.BigToHash(cancelMigrationBucket.BucketId.BigInt()),
		Operator:   common.HexToAddress(cancelMigrationBucket.Operator),
		BucketName: cancelMigrationBucket.BucketName,

		CreateAt:     block.Block.Height,
		CreateTxHash: txHash,
		CreateTime:   block.Block.Time.UTC().Unix(),
	}

	return m.db.SaveEventCancelMigrationBucket(ctx, eventCancelMigrationBucket)
}
