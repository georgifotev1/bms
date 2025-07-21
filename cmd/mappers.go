package main

import "github.com/georgifotev1/bms/internal/store"

// Mappers
func userResponseMapper(user *store.User) UserResponse {
	return UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Avatar:    user.Avatar.String,
		Verified:  user.Verified,
		BrandId:   user.BrandID.Int32,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}
func brandResponseMapper(brand *store.Brand, links []*store.BrandSocialLink, hours []*store.BrandWorkingHour) store.BrandResponse {
	socialLinks := []store.SocialLink{}
	workingHours := []store.WorkingHour{}

	if links != nil {
		for _, link := range links {
			socialLinks = append(socialLinks, store.SocialLink{
				BrandID:     link.BrandID,
				Platform:    link.Platform,
				Url:         link.Url,
				DisplayName: link.DisplayName.String,
				CreatedAt:   link.CreatedAt,
				UpdatedAt:   link.UpdatedAt,
			})
		}
	}

	if hours != nil {
		for _, hour := range hours {
			openTime := hour.OpenTime.Time.Format("15:04")
			closeTime := hour.CloseTime.Time.Format("15:04")

			workingHour := store.WorkingHour{
				ID:        hour.ID,
				BrandID:   hour.BrandID,
				DayOfWeek: hour.DayOfWeek,
				OpenTime:  openTime,
				CloseTime: closeTime,
				IsClosed:  hour.IsClosed,
				CreatedAt: hour.CreatedAt,
				UpdatedAt: hour.UpdatedAt,
			}
			workingHours = append(workingHours, workingHour)
		}
	}

	return store.BrandResponse{
		ID:           brand.ID,
		Name:         brand.Name,
		PageUrl:      brand.PageUrl,
		Description:  brand.Description.String,
		Email:        brand.Email.String,
		Phone:        brand.Phone.String,
		Country:      brand.Country.String,
		State:        brand.State.String,
		ZipCode:      brand.ZipCode.String,
		City:         brand.City.String,
		Address:      brand.Address.String,
		LogoUrl:      brand.LogoUrl.String,
		BannerUrl:    brand.BannerUrl.String,
		Currency:     brand.Currency.String,
		CreatedAt:    brand.CreatedAt,
		UpdatedAt:    brand.UpdatedAt,
		SocialLinks:  socialLinks,
		WorkingHours: workingHours,
	}
}

func serviceResponseMapper(service *store.Service, providers []int64) ServiceResponse {
	return ServiceResponse{
		ID:          service.ID,
		Title:       service.Title,
		Description: service.Description.String,
		Duration:    service.Duration,
		BufferTime:  service.BufferTime.Int32,
		Cost:        service.Cost.String,
		IsVisible:   service.IsVisible,
		ImageUrl:    service.ImageUrl.String,
		BrandID:     service.BrandID,
		Providers:   providers,
		CreatedAt:   service.CreatedAt,
		UpdatedAt:   service.UpdatedAt,
	}
}

func customerResponseMapper(customer *store.Customer) CustomerResponse {
	return CustomerResponse{
		ID:          customer.ID,
		Name:        customer.Name,
		Email:       customer.Email.String,
		BrandId:     customer.BrandID,
		PhoneNumber: customer.PhoneNumber,
	}
}

func customersResponseMapper(customer *store.Customer) CustomerResponse {
	return CustomerResponse{
		ID:          customer.ID,
		Name:        customer.Name,
		Email:       customer.Email.String,
		BrandId:     customer.BrandID,
		PhoneNumber: customer.PhoneNumber,
	}
}

func eventResponseMapper(event *store.Event) EventResponse {
	return EventResponse{
		ID:           event.ID,
		CustomerID:   event.CustomerID,
		ServiceID:    event.ServiceID,
		UserID:       event.UserID,
		BrandID:      event.BrandID,
		StartTime:    event.StartTime,
		EndTime:      event.EndTime,
		CustomerName: event.CustomerName,
		UserName:     event.UserName,
		ServiceName:  event.ServiceName,
		BufferTime:   event.BufferTime.Int32,
		Cost:         event.Cost.String,
		Comment:      event.Comment.String,
		CreatedAt:    event.CreatedAt,
		UpdatedAt:    event.UpdatedAt,
	}
}
