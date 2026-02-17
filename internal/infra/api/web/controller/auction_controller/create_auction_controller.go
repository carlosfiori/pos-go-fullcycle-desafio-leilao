package auction_controller

import (
	"context"
	"net/http"

	"github.com/carlosfiori/pos-go-fullcycle-desafio-leilao/configuration/rest_err"
	"github.com/carlosfiori/pos-go-fullcycle-desafio-leilao/internal/infra/api/web/validation"
	"github.com/carlosfiori/pos-go-fullcycle-desafio-leilao/internal/usecase/auction_usecase"
	"github.com/gin-gonic/gin"
)

type AuctionController struct {
	auctionUseCase auction_usecase.AuctionUseCaseInterface
}

func NewAuctionController(auctionUseCase auction_usecase.AuctionUseCaseInterface) *AuctionController {
	return &AuctionController{
		auctionUseCase: auctionUseCase,
	}
}

func (u *AuctionController) CreateAuction(c *gin.Context) {
	var auctionInputDTO auction_usecase.AuctionInputDTO

	if err := c.ShouldBindJSON(&auctionInputDTO); err != nil {
		restErr := validation.ValidateErr(err)

		c.JSON(restErr.Code, restErr)
		return
	}

	err := u.auctionUseCase.CreateAuction(context.Background(), auctionInputDTO)
	if err != nil {
		restErr := rest_err.ConvertError(err)

		c.JSON(restErr.Code, restErr)
		return
	}

	c.Status(http.StatusCreated)
}
