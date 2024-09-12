/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package slack

import (
	"time"

	"github.com/segmentfault/pacman/log"
)

func (uc *UserCenter) CronSyncData() {
	go func() {
		ticker := time.NewTicker(time.Hour) // 每小时触发一次
		defer ticker.Stop()                 // 停止ticker，防止泄露

		for {
			select {
			case <-ticker.C:
				log.Infof("UserCenter is syncing Slack user data...")
				uc.syncSlackUsers() // 调用同步Slack用户的函数
			}
		}
	}()
}
