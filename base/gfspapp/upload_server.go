package gfspapp

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bnb-chain/greenfield-storage-provider/base/types/gfsperrors"
	"github.com/bnb-chain/greenfield-storage-provider/base/types/gfspserver"
	"github.com/bnb-chain/greenfield-storage-provider/base/types/gfsptask"
	"github.com/bnb-chain/greenfield-storage-provider/core/rcmgr"
	"github.com/bnb-chain/greenfield-storage-provider/pkg/log"
	"github.com/bnb-chain/greenfield-storage-provider/pkg/metrics"
)

var (
	ErrUploadObjectDangling  = gfsperrors.Register(BaseCodeSpace, http.StatusBadRequest, 991101, "OoooH... request lost")
	ErrUploadExhaustResource = gfsperrors.Register(BaseCodeSpace, http.StatusBadRequest, 991102, "server overload, try again later")
	ErrExceptionsStream      = gfsperrors.Register(BaseCodeSpace, http.StatusBadRequest, 991103, "stream closed abnormally")
)

var _ gfspserver.GfSpUploadServiceServer = &GfSpBaseApp{}

func (g *GfSpBaseApp) GfSpUploadObject(stream gfspserver.GfSpUploadService_GfSpUploadObjectServer) error {
	var (
		span          rcmgr.ResourceScopeSpan
		task          *gfsptask.GfSpUploadObjectTask
		req           *gfspserver.GfSpUploadObjectRequest
		resp          = &gfspserver.GfSpUploadObjectResponse{}
		pRead, pWrite = io.Pipe()
		initCh        = make(chan struct{})
		ctx, cancel   = context.WithCancel(context.Background())
		err           error
		receiveSize   int
	)
	startTime := time.Now()
	defer func() {
		defer cancel()
		if span != nil {
			span.Done()
		}
		if task != nil {
			g.uploader.PostUploadObject(ctx, task)
			log.CtxDebugw(ctx, "finish to receive object stream data", "info", task.Info(),
				"receive_size", receiveSize, "error", err)
		} else {
			log.CtxDebugw(ctx, "finish to receive object stream data",
				"receive_size", receiveSize, "error", err)
		}
		if err != nil {
			resp.Err = gfsperrors.MakeGfSpError(err)
			metrics.ReqCounter.WithLabelValues(UploaderFailurePutObject).Inc()
			metrics.ReqTime.WithLabelValues(UploaderFailurePutObject).Observe(time.Since(startTime).Seconds())
		} else {
			metrics.ReqCounter.WithLabelValues(UploaderSuccessPutObject).Inc()
			metrics.ReqTime.WithLabelValues(UploaderSuccessPutObject).Observe(time.Since(startTime).Seconds())
		}

		err = stream.SendAndClose(resp)
		if err != nil {
			log.CtxErrorw(ctx, "failed to close upload object stream", "error", err)
		}
	}()

	go func() {
		init := false
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			req, err = stream.Recv()
			if err == io.EOF {
				if len(req.GetPayload()) != 0 {
					pWrite.Write(req.GetPayload())
				}
				log.CtxDebugw(ctx, "received last upload stream data")
				err = nil
				pWrite.Close()
				return
			}
			if err != nil {
				log.CtxErrorw(ctx, "failed to receive object ", "error", err)
				err = ErrExceptionsStream
				pWrite.CloseWithError(err)
				return
			}
			if !init {
				init = true
				task = req.GetUploadObjectTask()
				if task == nil {
					log.CtxErrorw(ctx, "[BUG] failed to receive object, upload object task pointer dangling !!!")
					err = ErrUploadObjectDangling
					pWrite.CloseWithError(err)
					return
				}
				ctx = log.WithValue(ctx, log.CtxKeyTask, task.Key().String())
				span, err = g.uploader.ReserveResource(ctx, task.EstimateLimit().ScopeStat())
				if err != nil {
					log.CtxErrorw(ctx, "failed to reserve resource", "error", err)
					err = ErrUploadExhaustResource
					pWrite.CloseWithError(err)
					return
				}
				err = g.uploader.PreUploadObject(ctx, task)
				task.AppendLog(fmt.Sprintf("uploader-prepare-upload-task-cost:%d", time.Now().UnixMilli()-startTime.UnixMilli()))
				if err != nil {
					log.CtxErrorw(ctx, "failed to pre upload object", "error", err)
					pWrite.CloseWithError(err)
					return
				}
				initCh <- struct{}{}
			}
			receiveSize += len(req.GetPayload())
			pWrite.Write(req.GetPayload())
		}
	}()

	select {
	case <-ctx.Done():
		return nil
	case <-initCh:
		log.CtxDebugw(ctx, "received first upload stream data")
	}
	err = g.uploader.HandleUploadObjectTask(ctx, task, pRead)
	if err != nil {
		log.CtxErrorw(ctx, "failed to upload object data", "error", err)
		pWrite.CloseWithError(err)
		return nil
	}
	log.CtxDebugw(ctx, "succeed to upload object")
	return nil
}

func (g *GfSpBaseApp) GfSpResumableUploadObject(stream gfspserver.GfSpUploadService_GfSpResumableUploadObjectServer) error {
	var (
		span          rcmgr.ResourceScopeSpan
		task          *gfsptask.GfSpResumableUploadObjectTask
		req           *gfspserver.GfSpResumableUploadObjectRequest
		resp          = &gfspserver.GfSpResumableUploadObjectResponse{}
		pRead, pWrite = io.Pipe()
		initCh        = make(chan struct{})
		ctx, cancel   = context.WithCancel(context.Background())
		err           error
		receiveSize   int
	)
	defer func() {
		defer cancel()
		if span != nil {
			span.Done()
		}
		if task != nil {
			g.uploader.PostResumableUploadObject(ctx, task)
			log.CtxDebugw(ctx, "finish to receive object stream data", "info", task.Info(),
				"receive_size", receiveSize, "error", err)
		} else {
			log.CtxDebugw(ctx, "finish to receive object stream data",
				"receive_size", receiveSize, "error", err)
		}
		if err != nil {
			resp.Err = gfsperrors.MakeGfSpError(err)
		}

		err = stream.SendAndClose(resp)
		if err != nil {
			log.CtxErrorw(ctx, "failed to close upload object stream", "error", err)
		}
	}()

	go func() {
		init := false
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			req, err = stream.Recv()
			if err == io.EOF {
				if len(req.GetPayload()) != 0 {
					pWrite.Write(req.GetPayload())
				}
				log.CtxDebugw(ctx, "received last upload stream data")
				err = nil
				pWrite.Close()
				return
			}
			if err != nil {
				log.CtxErrorw(ctx, "failed to receive object ", "error", err)
				err = ErrExceptionsStream
				pWrite.CloseWithError(err)
				return
			}
			if !init {
				init = true
				task = req.GetResumableUploadObjectTask()
				if task == nil {
					log.CtxErrorw(ctx, "[BUG] failed to receive object, upload object task pointer dangling !!!")
					err = ErrUploadObjectDangling
					pWrite.CloseWithError(err)
					return
				}
				ctx = log.WithValue(ctx, log.CtxKeyTask, task.Key().String())
				span, err = g.uploader.ReserveResource(ctx, task.EstimateLimit().ScopeStat())
				if err != nil {
					log.CtxErrorw(ctx, "failed to reserve resource", "error", err)
					err = ErrUploadExhaustResource
					pWrite.CloseWithError(err)
					return
				}
				err = g.uploader.PreResumableUploadObject(ctx, task)
				if err != nil {
					log.CtxErrorw(ctx, "failed to pre upload object", "error", err)
					pWrite.CloseWithError(err)
					return
				}
				initCh <- struct{}{}
			}
			receiveSize += len(req.GetPayload())
			pWrite.Write(req.GetPayload())
		}
	}()

	select {
	case <-ctx.Done():
		return nil
	case <-initCh:
		log.CtxDebugw(ctx, "received first upload stream data")
	}
	err = g.uploader.HandleResumableUploadObjectTask(ctx, task, pRead)
	if err != nil {
		log.CtxErrorw(ctx, "failed to upload object data", "error", err)
		pWrite.CloseWithError(err)
		return nil
	}
	log.CtxDebugw(ctx, "succeed to upload object")
	return nil
}
