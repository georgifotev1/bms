package main

import (
	"fmt"
	"net/http"

	"github.com/cloudinary/cloudinary-go/v2/api/admin"
)

//	@Summary		Get the images of the brand from bucket
//	@Description	Retrieves a brand's stored images.
//	@Tags			admin
//	@Produce		json
//	@Success		200	{object}	[]string	"Brand images"
//	@Failure		400	{object}	error		"Bad request - Invalid brand ID"
//	@Failure		500	{object}	error		"Internal server error"
//	@Router			/admin/images [get]
func (app *application) getImagesHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctxUser := getUserFromCtx(ctx)

	images, err := app.imageService.Admin.AssetsByAssetFolder(ctx, admin.AssetsByAssetFolderParams{
		AssetFolder: fmt.Sprintf("brand-%d", ctxUser.BrandID.Int32),
	})
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	response := brandImagesResponseMapper(images)

	if err := writeJSON(w, http.StatusOK, response); err != nil {
		app.internalServerError(w, r, err)
	}
}
