package api

import (
	"github.com/grassrootseconomics/eth-custodial/pkg/api"
)

type Pagination struct {
	api.OTXByAccountRequest

	FirstPage bool
}

func validatePagination(q api.OTXByAccountRequest) Pagination {
	var firstPage = false

	if q.PerPage > 100 {
		q.PerPage = 100
	}

	if !q.Next && q.Cursor < 1 {
		firstPage = true
	}

	return Pagination{
		OTXByAccountRequest: q,
		FirstPage:           firstPage,
	}
}
