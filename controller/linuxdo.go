// Copyright (c) 2025 Tethys Plex
//
// This file is part of Veloera.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.
package controller

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"veloera/common"
	"veloera/model"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

type LinuxdoUser struct {
	Id         int    `json:"id"`
	Username   string `json:"username"`
	Name       string `json:"name"`
	Active     bool   `json:"active"`
	TrustLevel int    `json:"trust_level"`
	Silenced   bool   `json:"silenced"`
}

func LinuxDoBind(c *gin.Context) {
	if !common.LinuxDOOAuthEnabled {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "管理员未开启通过 Linux DO 登录以及注册",
		})
		return
	}

	code := c.Query("code")
	linuxdoUser, err := getLinuxdoUserInfoByCode(code, c)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	user := model.User{
		LinuxDOId: strconv.Itoa(linuxdoUser.Id),
	}

	if model.IsLinuxDOIdAlreadyTaken(user.LinuxDOId) {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "该 Linux DO 账户已被绑定",
		})
		return
	}

	session := sessions.Default(c)
	id := session.Get("id")
	user.Id = id.(int)

	err = user.FillUserById()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	user.LinuxDOId = strconv.Itoa(linuxdoUser.Id)
	err = user.Update(false)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "bind",
	})
}

func getLinuxdoUserInfoByCode(code string, c *gin.Context) (*LinuxdoUser, error) {
	if code == "" {
		return nil, errors.New("invalid code")
	}

	// Get access token using Basic auth
	tokenEndpoint := "https://connect.linux.do/oauth2/token"
	credentials := common.LinuxDOClientId + ":" + common.LinuxDOClientSecret
	basicAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(credentials))

	// Get redirect URI from request
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	redirectURI := fmt.Sprintf("%s://%s/api/oauth/linuxdo", scheme, c.Request.Host)

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)

	req, err := http.NewRequest("POST", tokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", basicAuth)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := http.Client{Timeout: 5 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return nil, errors.New("failed to connect to Linux DO server")
	}
	defer res.Body.Close()

	var tokenRes struct {
		AccessToken string `json:"access_token"`
		Message     string `json:"message"`
	}
	if err := json.NewDecoder(res.Body).Decode(&tokenRes); err != nil {
		return nil, err
	}

	if tokenRes.AccessToken == "" {
		return nil, fmt.Errorf("failed to get access token: %s", tokenRes.Message)
	}

	// Get user info
	userEndpoint := "https://connect.linux.do/api/user"
	req, err = http.NewRequest("GET", userEndpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+tokenRes.AccessToken)
	req.Header.Set("Accept", "application/json")

	res2, err := client.Do(req)
	if err != nil {
		return nil, errors.New("failed to get user info from Linux DO")
	}
	defer res2.Body.Close()

	var linuxdoUser LinuxdoUser
	if err := json.NewDecoder(res2.Body).Decode(&linuxdoUser); err != nil {
		return nil, err
	}

	if linuxdoUser.Id == 0 {
		return nil, errors.New("invalid user info returned")
	}

	return &linuxdoUser, nil
}

func LinuxdoOAuth(c *gin.Context) {
	session := sessions.Default(c)

	errorCode := c.Query("error")
	if errorCode != "" {
		errorDescription := c.Query("error_description")
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": errorDescription,
		})
		return
	}

	state := c.Query("state")
	if state == "" || session.Get("oauth_state") == nil || state != session.Get("oauth_state").(string) {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "state is empty or not same",
		})
		return
	}

	username := session.Get("username")
	if username != nil {
		LinuxDoBind(c)
		return
	}

	if !common.LinuxDOOAuthEnabled {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "管理员未开启通过 Linux DO 登录以及注册",
		})
		return
	}

	code := c.Query("code")
	linuxdoUser, err := getLinuxdoUserInfoByCode(code, c)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	user := model.User{
		LinuxDOId: strconv.Itoa(linuxdoUser.Id),
	}

	// Check if user exists
	if model.IsLinuxDOIdAlreadyTaken(user.LinuxDOId) {
		err := user.FillUserByLinuxDOId()
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
		if user.Id == 0 {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "用户已注销",
			})
			return
		}
	} else {
		if common.RegisterEnabled {
			user.DisplayName = linuxdoUser.Name
			user.Role = common.RoleCommonUser
			user.Status = common.UserStatusEnabled

			affCode := session.Get("aff")
			inviterId := 0
			if affCode != nil {
				inviterId, _ = model.GetUserIdByAffCode(affCode.(string))
			}

			// Try to insert user, handle username uniqueness constraint
			var err error
			baseUserId := model.GetMaxUserId() + 1
			for i := 0; i < 5; i++ {
				user.Username = "linuxdo_" + strconv.Itoa(baseUserId+i)
				err = user.Insert(inviterId)
				if err == nil {
					break
				}
				// Check if error is about username uniqueness
				if strings.Contains(err.Error(), "UNIQUE constraint failed: users.username") {
					continue
				}
				// If it's another error, break the loop
				break
			}

			if err != nil {
				c.JSON(http.StatusOK, gin.H{
					"success": false,
					"message": err.Error(),
				})
				return
			}
		} else {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "管理员关闭了新用户注册",
			})
			return
		}
	}

	if user.Status != common.UserStatusEnabled {
		c.JSON(http.StatusOK, gin.H{
			"message": "用户已被封禁",
			"success": false,
		})
		return
	}

	setupLogin(&user, c)
}
