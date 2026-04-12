// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package app

import (
	"net/http"

	"github.com/mattermost/mattermost/server/public/model"
)

const (
	maxUsersLimit     = 200
	maxUsersHardLimit = 250
)

func (a *App) GetServerLimits() (*model.ServerLimits, *model.AppError) {
	limits := &model.ServerLimits{}

	// User limits are disabled for self-hosted: always 0 (unlimited).
	limits.MaxUsersLimit = 0
	limits.MaxUsersHardLimit = 0

	// Post history limits are disabled for self-hosted: all posts are always accessible.
	// limits.PostHistoryLimit and limits.LastAccessiblePostTime remain at 0 (no limit).

	activeUserCount, appErr := a.Srv().Store().User().Count(model.UserCountOptions{})
	if appErr != nil {
		return nil, model.NewAppError("GetServerLimits", "app.limits.get_app_limits.user_count.store_error", nil, "", http.StatusInternalServerError).Wrap(appErr)
	}

	if a.shouldTrackSingleChannelGuests() {
		singleChannelGuestCount, err := a.Srv().Store().User().AnalyticsGetSingleChannelGuestCount()
		if err != nil {
			return nil, model.NewAppError("GetServerLimits", "app.limits.get_app_limits.single_channel_guest_count.store_error", nil, "", http.StatusInternalServerError).Wrap(err)
		}

		// Single-channel guests are free and excluded from the primary seat count.
		limits.ActiveUserCount = max(activeUserCount-singleChannelGuestCount, 0)
		limits.SingleChannelGuestCount = singleChannelGuestCount
		// Guests are unlimited.
		limits.SingleChannelGuestLimit = 0
	} else {
		limits.ActiveUserCount = activeUserCount
	}

	return limits, nil
}

func (a *App) shouldTrackSingleChannelGuests() bool {
	license := a.License()
	if license == nil {
		return false
	}
	if license.IsMattermostEntry() {
		return false
	}
	cfg := a.Config()
	if cfg == nil || cfg.GuestAccountsSettings.Enable == nil {
		return false
	}

	return *cfg.GuestAccountsSettings.Enable
}

func (a *App) GetPostHistoryLimit() int64 {
	// Post history limits are disabled for self-hosted: always return 0 (unlimited).
	return 0
}

func (a *App) isAtUserLimit() (bool, *model.AppError) {
	userLimits, appErr := a.GetServerLimits()
	if appErr != nil {
		return false, appErr
	}

	if userLimits.MaxUsersHardLimit == 0 {
		return false, nil
	}

	return userLimits.ActiveUserCount >= userLimits.MaxUsersHardLimit, appErr
}
