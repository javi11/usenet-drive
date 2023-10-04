package handlers

import (
	"encoding/json"
	"net/http"

	corruptednzbsmanager "github.com/javi11/usenet-drive/internal/usenet/corrupted-nzbs-manager"
	"github.com/javi11/usenet-drive/internal/utils"
	echo "github.com/labstack/echo/v4"
)

type Filter struct {
	ID    string `query:"id"`
	Value string `query:"value"`
}

type FilterModes struct {
	Path      utils.FilterMode `query:"path"`
	CreatedAt utils.FilterMode `query:"created_at"`
	Error     utils.FilterMode `query:"error"`
}

type Sorting struct {
	ID   string `query:"id"`
	Desc bool   `query:"desc"`
}

type QueryParams struct {
	Filters     string `query:"filters"`
	FilterModes string `query:"filterModes"`
	Sorting     string `query:"sorting"`
	Offset      int    `query:"offset"`
	Limit       int    `query:"limit"`
}

func GetCorruptedNzbListHandler(cNzb corruptednzbsmanager.CorruptedNzbsManager) echo.HandlerFunc {
	return func(c echo.Context) error {
		limit := 10
		offset := 0

		queryParams := new(QueryParams)
		if err := c.Bind(queryParams); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		if queryParams.Limit != 0 {
			limit = queryParams.Limit
		}

		if queryParams.Offset != 0 {
			offset = queryParams.Offset
		}

		filter := []Filter{}
		if err := json.Unmarshal([]byte(queryParams.Filters), &filter); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		filterMode := FilterModes{}
		if err := json.Unmarshal([]byte(queryParams.FilterModes), &filterMode); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		filters := &corruptednzbsmanager.Filters{}
		for _, filter := range filter {
			switch filter.ID {
			case "path":
				filters.Path = utils.Filter{
					Value: filter.Value,
					Mode:  filterMode.Path,
				}
			case "created_at":
				filters.CreatedAt = utils.Filter{
					Value: filter.Value,
					Mode:  filterMode.CreatedAt,
				}
			case "error":
				filters.Error = utils.Filter{
					Value: filter.Value,
					Mode:  filterMode.Error,
				}
			}
		}

		sorting := []Sorting{}
		if err := json.Unmarshal([]byte(queryParams.Sorting), &sorting); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		sortBy := &corruptednzbsmanager.SortBy{}
		for _, sorting := range sorting {
			switch sorting.ID {
			case "path":
				sortBy.Path = utils.SortByDirectionAsc
				if sorting.Desc {
					sortBy.Path = utils.SortByDirectionDesc
				}
			case "created_at":
				sortBy.CreatedAt = utils.SortByDirectionAsc
				if sorting.Desc {
					sortBy.CreatedAt = utils.SortByDirectionDesc
				}
			case "error":
				sortBy.Error = utils.SortByDirectionAsc
				if sorting.Desc {
					sortBy.Error = utils.SortByDirectionDesc
				}
			}
		}

		result, err := cNzb.List(c.Request().Context(), limit, offset, filters, sortBy)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		return c.JSON(http.StatusOK, result)
	}
}
