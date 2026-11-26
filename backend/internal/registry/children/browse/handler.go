package browse

import (
	"context"

	"github.com/danielgtaylor/huma/v2"

	"getarcane.dev/arcane/v3/internal/common"
	"getarcane.dev/arcane/v3/pkg/handlerutil"
	"getarcane.dev/types/v3"
)

type (
	Handler struct {
		service *Service
	}

	ListRegistryRepositoriesInput struct {
		ID        string `path:"id" doc:"Registry ID"`
		Namespace string `query:"namespace" doc:"Namespace/project to browse"`
		Search    string `query:"search" doc:"Search query"`
		Sort      string `query:"sort" doc:"Column to sort by"`
		Order     string `query:"order" default:"asc" doc:"Sort direction"`
		Start     int    `query:"start" default:"0" doc:"Start index"`
		Limit     int    `query:"limit" default:"20" doc:"Items per page"`
	}

	ListRegistryTagsInput struct {
		ID         string `path:"id" doc:"Registry ID"`
		Repository string `query:"repository" required:"true" minLength:"1" doc:"Repository name"`
		Search     string `query:"search" doc:"Search query"`
		Sort       string `query:"sort" doc:"Column to sort by"`
		Order      string `query:"order" default:"asc" doc:"Sort direction"`
		Start      int    `query:"start" default:"0" doc:"Start index"`
		Limit      int    `query:"limit" default:"20" doc:"Items per page"`
	}

	RegistryScanReportInput struct {
		ID         string `path:"id" doc:"Registry ID"`
		Repository string `query:"repository" required:"true" minLength:"1" doc:"Project and repository"`
		Digest     string `query:"digest" required:"true" minLength:"1" doc:"Image manifest digest"`
		Platform   string `query:"platform" doc:"Optional platform, such as linux/amd64"`
	}

	DeleteRegistryTagInput struct {
		ID         string `path:"id" doc:"Registry ID"`
		Repository string `query:"repository" required:"true" minLength:"1" doc:"Repository name"`
		Tag        string `query:"tag" required:"true" minLength:"1" doc:"Tag to delete"`
	}
)

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ListNamespaces(ctx context.Context, input *ListRegistryRepositoriesInput) (*handlerutil.Page[types.ContainerRegistryNamespace], error) {
	params := handlerutil.PaginationParams(input.Start, input.Limit, input.Sort, input.Order, input.Search)
	items, paging, err := h.service.ListNamespaces(ctx, input.ID, params)
	if err != nil {
		apiErr := common.ToAPIError(err)
		return nil, huma.NewError(apiErr.HTTPStatus(), "Failed to discover namespaces: "+err.Error())
	}
	return &handlerutil.Page[types.ContainerRegistryNamespace]{
		Body: types.Paginated[types.ContainerRegistryNamespace]{
			Success:    true,
			Data:       items,
			Pagination: handlerutil.PaginationResponse(paging),
		},
	}, nil
}

func (h *Handler) ListRepositories(ctx context.Context, input *ListRegistryRepositoriesInput) (*handlerutil.Page[types.ContainerRegistryRepository], error) {
	params := handlerutil.PaginationParams(input.Start, input.Limit, input.Sort, input.Order, input.Search)

	repositories, paginationResp, err := h.service.ListRepositories(ctx, input.ID, input.Namespace, params)
	if err != nil {
		apiErr := common.ToAPIError(err)
		return nil, huma.NewError(apiErr.HTTPStatus(), "Failed to list repositories: "+err.Error())
	}

	return &handlerutil.Page[types.ContainerRegistryRepository]{
		Body: types.Paginated[types.ContainerRegistryRepository]{
			Success:    true,
			Data:       repositories,
			Pagination: handlerutil.PaginationResponse(paginationResp),
		},
	}, nil
}

func (h *Handler) ListTags(ctx context.Context, input *ListRegistryTagsInput) (*handlerutil.Page[types.ContainerRegistryRepositoryTag], error) {
	params := handlerutil.PaginationParams(input.Start, input.Limit, input.Sort, input.Order, input.Search)

	tags, paginationResp, err := h.service.ListRepositoryTags(ctx, input.ID, input.Repository, params)
	if err != nil {
		apiErr := common.ToAPIError(err)
		return nil, huma.NewError(apiErr.HTTPStatus(), "Failed to list tags: "+err.Error())
	}

	return &handlerutil.Page[types.ContainerRegistryRepositoryTag]{
		Body: types.Paginated[types.ContainerRegistryRepositoryTag]{
			Success:    true,
			Data:       tags,
			Pagination: handlerutil.PaginationResponse(paginationResp),
		},
	}, nil
}

func (h *Handler) DeleteTag(ctx context.Context, input *DeleteRegistryTagInput) (*handlerutil.Out[types.ContainerRegistryDeleteTagResponse], error) {
	digest, err := h.service.DeleteRepositoryTag(ctx, input.ID, input.Repository, input.Tag)
	if err != nil {
		apiErr := common.ToAPIError(err)
		return nil, huma.NewError(apiErr.HTTPStatus(), "Failed to delete tag: "+err.Error())
	}

	return &handlerutil.Out[types.ContainerRegistryDeleteTagResponse]{
		Body: types.ApiResponse[types.ContainerRegistryDeleteTagResponse]{
			Success: true,
			Data:    types.ContainerRegistryDeleteTagResponse{Digest: digest},
		},
	}, nil
}

func (h *Handler) ScanReport(ctx context.Context, input *RegistryScanReportInput) (*handlerutil.Out[types.RegistryScanReport], error) {
	report, err := h.service.ScanArtifact(ctx, types.RegistryArtifactScanOptions{RegistryID: input.ID, Repository: input.Repository, Digest: input.Digest, Platform: input.Platform})
	if err != nil {
		apiErr := common.ToAPIError(err)
		return nil, huma.NewError(apiErr.HTTPStatus(), err.Error())
	}
	return &handlerutil.Out[types.RegistryScanReport]{
		Body: types.ApiResponse[types.RegistryScanReport]{Success: true, Data: report},
	}, nil
}
