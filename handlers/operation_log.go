/**
 * Tencent is pleased to support the open source community by making Polaris available.
 *
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 *
 * Licensed under the BSD 3-Clause License (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://opensource.org/licenses/BSD-3-Clause
 *
 * Unless required by applicable law or agreed to in writing, software distributed
 * under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
 * CONDITIONS OF ANY KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations under the License.
 */

package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/polarismesh/polaris-console/bootstrap"
	httpcommon "github.com/polarismesh/polaris-console/common/http"
	"github.com/polarismesh/polaris-console/common/model"
	"github.com/polarismesh/polaris-console/common/operation"
)

var (
	_searchOperationLogParams = map[string]struct{}{
		"namespace":        {},
		"resource_type":    {},
		"resource_name":    {},
		"operation_type":   {},
		"operator":         {},
		"operation_detail": {},
		"start_time":       {},
		"end_time":         {},
		"limit":            {},
		"offset":           {},
		"extend_info":      {},
	}
)

type OperationLogResponse struct {
	Code       uint32            `json:"code"`
	Info       string            `json:"info"`
	Total      uint64            `json:"total"`
	Size       uint32            `json:"size"`
	HasNext    bool              `json:"has_next"`
	Data       []OperationRecord `json:"data"`
	ExtendInfo string            `json:"extend_info"`
}

type OperationRecord struct {
	ResourceType    string `json:"resource_type"`
	ResourceName    string `json:"resource_name"`
	Namespace       string `json:"namespace"`
	OperationType   string `json:"operation_type"`
	Operator        string `json:"operator"`
	OperationDetail string `json:"operation_detail"`
	HappenTime      string `json:"happen_time"`
}

func DescribeOperationHistoryLog(conf *bootstrap.Config) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if !verifyAccessPermission(ctx, conf) {
			return
		}
		if !conf.HasFutures(model.FutureLogObservability) || len(conf.OperationServer.RequestURL) == 0 {
			ctx.JSON(http.StatusOK, model.QueryResponse{
				Code:     200000,
				Size:     0,
				Amount:   0,
				Data:     []OperationRecord{},
				HashNext: false,
			})
			return
		}
		reader, err := GetHistoryLogReader(conf)
		if err != nil {
			ctx.JSON(http.StatusNotFound, model.Response{
				Code: 400404,
				Info: err.Error(),
			})
			return
		}
		if reader == nil {
			ctx.JSON(http.StatusOK, model.QueryResponse{
				Code:     200000,
				Size:     0,
				Amount:   0,
				Data:     []OperationRecord{},
				HashNext: false,
			})
			return
		}

		filters := httpcommon.ParseQueryParams(ctx.Request)
		param, err := parseHttpQueryToSearchParams(filters, _searchOperationLogParams)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, model.Response{
				Code: 400000,
				Info: err.Error(),
			})
			return
		}

		resp := &OperationLogResponse{}
		if err := reader.Query(context.Background(), param, resp); err != nil {
			ctx.JSON(http.StatusInternalServerError, model.Response{
				Code: 500000,
				Info: err.Error(),
			})
			return
		}

		ctx.JSON(http.StatusOK, resp)
		return
	}
}

// DescribeOperationTypes describe operation type desc list
func DescribeOperationTypes(conf *bootstrap.Config) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ret := make([]TypeInfo, 0, len(operation.OperationTypeInfos))
		for k, v := range operation.OperationTypeInfos {
			ret = append(ret, TypeInfo{
				Type: string(k),
				Desc: v,
			})
		}

		ctx.JSON(http.StatusOK, model.Response{
			Code: 200000,
			Info: "success",
			Data: ret,
		})
	}
}

// DescribeOperationResourceTypes describe operation type desc list
func DescribeOperationResourceTypes(conf *bootstrap.Config) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ret := make([]TypeInfo, 0, len(operation.ResourceTypeInfos))
		for k, v := range operation.ResourceTypeInfos {
			ret = append(ret, TypeInfo{
				Type: string(k),
				Desc: v,
			})
		}

		ctx.JSON(http.StatusOK, model.Response{
			Code: 200000,
			Info: "success",
			Data: ret,
		})
	}
}
