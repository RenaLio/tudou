package common

import (
	"mime/multipart"
	"net/http"

	"github.com/RenaLio/tudou/pkg/provider/types"
)

func CloneRequest(req *types.Request) *types.Request {
	return cloneRequest(req)
}

func cloneRequest(req *types.Request) *types.Request {
	if req == nil {
		return nil
	}
	cp := *req
	if req.Payload != nil {
		cp.Payload = append([]byte(nil), req.Payload...)
	}

	if req.Headers != nil {
		cp.Headers = req.Headers.Clone()
	} else {
		cp.Headers = make(http.Header)
	}

	if req.FormPayload != nil {
		fp := *req.FormPayload
		if req.FormPayload.Fields != nil {
			fp.Fields = make(map[string]string, len(req.FormPayload.Fields))
			for k, v := range req.FormPayload.Fields {
				fp.Fields[k] = v
			}
		}
		if req.FormPayload.Files != nil {
			fp.Files = make(map[string]*multipart.FileHeader, len(req.FormPayload.Files))
			for k, v := range req.FormPayload.Files {
				fp.Files[k] = v
			}
		}
		cp.FormPayload = &fp
	}
	return &cp
}

func CloneResponse(resp *types.Response) *types.Response {
	temp := *resp
	return &temp
}
