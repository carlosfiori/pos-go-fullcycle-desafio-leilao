package bid_controller

import (
	"context"
	"net/http"

	"github.com/carlosfiori/pos-go-fullcycle-desafio-leilao/configuration/rest_err"
	"github.com/carlosfiori/pos-go-fullcycle-desafio-leilao/internal/infra/api/web/validation"
	"github.com/carlosfiori/pos-go-fullcycle-desafio-leilao/internal/usecase/bid_usecase"
	"github.com/gin-gonic/gin"
)

type BidController struct {
	bidUseCase bid_usecase.BidUseCaseInterface
}

func NewBidController(bidUseCase bid_usecase.BidUseCaseInterface) *BidController {
	return &BidController{
		bidUseCase: bidUseCase,
	}
}

func (u *BidController) CreateBid(c *gin.Context) {
	var bidInputDTO bid_usecase.BidInputDTO

	if err := c.ShouldBindJSON(&bidInputDTO); err != nil {
		restErr := validation.ValidateErr(err)

		c.JSON(restErr.Code, restErr)
		return
	}

	err := u.bidUseCase.CreateBid(context.Background(), bidInputDTO)
	if err != nil {
		restErr := rest_err.ConvertError(err)

		c.JSON(restErr.Code, restErr)
		return
	}

	c.Status(http.StatusCreated)
}
