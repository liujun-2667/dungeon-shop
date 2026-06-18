package game

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"
	"time"

	"dungeon-shop/internal/models"
)

func getGuildMaxMembers(roomMaxPlayers int) int {
	return int(math.Ceil(float64(roomMaxPlayers) / 2.0))
}

func generateAbbreviation(name string) string {
	if len(name) <= 3 {
		return strings.ToUpper(name)
	}
	runes := []rune(name)
	if len(runes) <= 3 {
		return strings.ToUpper(string(runes))
	}
	return strings.ToUpper(string(runes[0]) + string(runes[1]) + string(runes[2]))
}

func (rm *RoomManager) CreateGuild(roomID, playerID, guildName string) (*models.Guild, string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	room, ok := rm.rooms[roomID]
	if !ok {
		return nil, "房间不存在"
	}

	player := room.Players[playerID]
	if player == nil {
		return nil, "玩家不存在"
	}

	if player.GuildID != "" {
		return nil, "你已经属于一个公会"
	}

	if player.Gold < models.GuildCreateCost {
		return nil, "金币不足，创建公会需要50金币"
	}

	for _, guild := range room.Guilds {
		if guild.Name == guildName {
			return nil, "公会名称已存在"
		}
	}

	maxMembers := getGuildMaxMembers(room.MaxPlayers)
	if maxMembers < 1 {
		maxMembers = 1
	}

	guild := &models.Guild{
		ID:            models.NewID(),
		RoomID:        roomID,
		Name:          guildName,
		Abbreviation:  generateAbbreviation(guildName),
		Level:         1,
		Treasury:      0,
		LeaderID:      playerID,
		Members:       make([]models.GuildMember, 0),
		Warehouse:     make([]models.Item, 0),
		CreatedAt:     time.Now().Unix(),
		BannedPlayers: make(map[string]int64),
		Tasks:         make([]models.GuildTask, 0),
		Logs:          make([]models.GuildLogEntry, 0),
		LastTaskWeek:  0,
	}

	guild.Members = append(guild.Members, models.GuildMember{
		PlayerID:     playerID,
		PlayerName:   player.Name,
		JoinTime:     time.Now().Unix(),
		JoinWeek:     room.CurrentWeek,
		IsLeader:     true,
		Contribution: 0,
		TotalExp:     0,
	})

	rm.addGuildLog(room, guild, playerID, player.Name, "创建了公会")

	player.Gold -= models.GuildCreateCost
	player.GuildID = guild.ID

	if room.Guilds == nil {
		room.Guilds = make(map[string]*models.Guild)
	}
	room.Guilds[guild.ID] = guild

	return guild, ""
}

func (rm *RoomManager) JoinGuild(roomID, playerID, guildID string) (*models.Guild, string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	room, ok := rm.rooms[roomID]
	if !ok {
		return nil, "房间不存在"
	}

	player := room.Players[playerID]
	if player == nil {
		return nil, "玩家不存在"
	}

	if player.GuildID != "" {
		return nil, "你已经属于一个公会"
	}

	guild, ok := room.Guilds[guildID]
	if !ok {
		return nil, "公会不存在"
	}

	maxMembers := getGuildMaxMembers(room.MaxPlayers)
	if len(guild.Members) >= maxMembers {
		return nil, "公会人数已满"
	}

	if kickedWeek, banned := guild.BannedPlayers[playerID]; banned {
		banDurationWeeks := int(models.GuildKickBanDuration)
		weeksSinceBan := room.CurrentWeek - int(kickedWeek)
		if weeksSinceBan < banDurationWeeks {
			remaining := banDurationWeeks - weeksSinceBan
			return nil, fmt.Sprintf("你被该公会踢出未满2周，还需等待%d周才能加入", remaining)
		}
		delete(guild.BannedPlayers, playerID)
	}

	guild.Members = append(guild.Members, models.GuildMember{
		PlayerID:     playerID,
		PlayerName:   player.Name,
		JoinTime:     time.Now().Unix(),
		JoinWeek:     room.CurrentWeek,
		IsLeader:     false,
		Contribution: 0,
		TotalExp:     0,
	})

	player.GuildID = guildID

	rm.addGuildLog(room, guild, playerID, player.Name, "加入了公会")
	rm.updateGuildTaskProgressLocked(room, guild, models.GuildTaskTypeMemberCount, 0)

	return guild, ""
}

func (rm *RoomManager) LeaveGuild(roomID, playerID string) string {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	room, ok := rm.rooms[roomID]
	if !ok {
		return "房间不存在"
	}

	player := room.Players[playerID]
	if player == nil || player.GuildID == "" {
		return "你没有加入任何公会"
	}

	guild, ok := room.Guilds[player.GuildID]
	if !ok {
		player.GuildID = ""
		return ""
	}

	rm.addGuildLog(room, guild, playerID, player.Name, "退出了公会")

	if len(guild.Members) == 1 {
		rm.distributeGuildWarehouseOnDisband(room, guild, playerID)
		delete(room.Guilds, guild.ID)
		player.GuildID = ""
		return ""
	}

	newLeaderName := ""
	if guild.LeaderID == playerID {
		newLeaderIdx := -1
		earliestJoinTime := int64(math.MaxInt64)
		for i, member := range guild.Members {
			if member.PlayerID != playerID && member.JoinTime < earliestJoinTime {
				earliestJoinTime = member.JoinTime
				newLeaderIdx = i
			}
		}

		if newLeaderIdx >= 0 {
			guild.Members[newLeaderIdx].IsLeader = true
			guild.LeaderID = guild.Members[newLeaderIdx].PlayerID
			newLeaderName = guild.Members[newLeaderIdx].PlayerName
		}
	}

	memberIdx := -1
	for i, member := range guild.Members {
		if member.PlayerID == playerID {
			memberIdx = i
			break
		}
	}
	if memberIdx >= 0 {
		guild.Members = append(guild.Members[:memberIdx], guild.Members[memberIdx+1:]...)
	}

	if newLeaderName != "" {
		rm.addGuildLog(room, guild, "", "系统", fmt.Sprintf("%s成为新会长", newLeaderName))
	}

	rm.updateGuildTaskProgressLocked(room, guild, models.GuildTaskTypeMemberCount, 0)

	player.GuildID = ""

	return ""
}

func (rm *RoomManager) KickMember(roomID, leaderID, targetPlayerID string) string {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	room, ok := rm.rooms[roomID]
	if !ok {
		return "房间不存在"
	}

	leader := room.Players[leaderID]
	if leader == nil || leader.GuildID == "" {
		return "你没有加入任何公会"
	}

	guild, ok := room.Guilds[leader.GuildID]
	if !ok {
		return "公会不存在"
	}

	if guild.LeaderID != leaderID {
		return "只有会长可以踢人"
	}

	if leaderID == targetPlayerID {
		return "不能踢自己"
	}

	targetPlayer := room.Players[targetPlayerID]
	if targetPlayer == nil || targetPlayer.GuildID != guild.ID {
		return "目标玩家不在该公会"
	}

	targetMemberName := targetPlayer.Name
	memberIdx := -1
	for i, member := range guild.Members {
		if member.PlayerID == targetPlayerID {
			memberIdx = i
			targetMemberName = member.PlayerName
			break
		}
	}
	if memberIdx >= 0 {
		guild.Members = append(guild.Members[:memberIdx], guild.Members[memberIdx+1:]...)
	}

	guild.BannedPlayers[targetPlayerID] = int64(room.CurrentWeek)
	targetPlayer.GuildID = ""

	rm.addGuildLog(room, guild, leaderID, leader.Name, fmt.Sprintf("踢出了成员%s", targetMemberName))
	rm.updateGuildTaskProgressLocked(room, guild, models.GuildTaskTypeMemberCount, 0)

	return ""
}

func (rm *RoomManager) UpgradeGuild(roomID, playerID string) (*models.Guild, string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	room, ok := rm.rooms[roomID]
	if !ok {
		return nil, "房间不存在"
	}

	player := room.Players[playerID]
	if player == nil || player.GuildID == "" {
		return nil, "你没有加入任何公会"
	}

	guild, ok := room.Guilds[player.GuildID]
	if !ok {
		return nil, "公会不存在"
	}

	if guild.LeaderID != playerID {
		return nil, "只有会长可以升级公会"
	}

	if guild.Level >= models.GuildMaxLevel {
		return nil, "公会已达到最高等级"
	}

	nextLevel := guild.Level + 1
	requiredGold, ok := models.GuildUpgradeRequirements[nextLevel]
	if !ok {
		return nil, "无法升级到该等级"
	}

	if guild.Treasury < requiredGold {
		return nil, "公会金库金币不足"
	}

	guild.Treasury -= requiredGold
	guild.Level = nextLevel

	rm.addGuildLog(room, guild, playerID, player.Name, fmt.Sprintf("将公会升级到Lv.%d", nextLevel))

	return guild, ""
}

func (rm *RoomManager) GetGuild(roomID, guildID string) (*models.Guild, bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	room, ok := rm.rooms[roomID]
	if !ok {
		return nil, false
	}

	guild, ok := room.Guilds[guildID]
	return guild, ok
}

func (rm *RoomManager) GetPlayerGuild(roomID, playerID string) (*models.Guild, bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	room, ok := rm.rooms[roomID]
	if !ok {
		return nil, false
	}

	player, ok := room.Players[playerID]
	if !ok || player.GuildID == "" {
		return nil, false
	}

	guild, ok := room.Guilds[player.GuildID]
	return guild, ok
}

func (rm *RoomManager) ListGuilds(roomID string) []*models.Guild {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	room, ok := rm.rooms[roomID]
	if !ok {
		return nil
	}

	guilds := make([]*models.Guild, 0, len(room.Guilds))
	for _, guild := range room.Guilds {
		guilds = append(guilds, guild)
	}
	return guilds
}

func (rm *RoomManager) DepositGuildWarehouse(roomID, playerID, itemID string) string {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	room, ok := rm.rooms[roomID]
	if !ok {
		return "房间不存在"
	}

	player := room.Players[playerID]
	if player == nil || player.GuildID == "" {
		return "你没有加入任何公会"
	}

	guild, ok := room.Guilds[player.GuildID]
	if !ok {
		return "公会不存在"
	}

	warehouseCapacity := guild.Level * models.GuildWarehousePerLevel
	if len(guild.Warehouse) >= warehouseCapacity {
		return "公会仓库已满"
	}

	itemIdx := -1
	for i, item := range player.Warehouse {
		if item.ID == itemID {
			itemIdx = i
			break
		}
	}
	if itemIdx == -1 {
		return "仓库中未找到该物品"
	}

	item := player.Warehouse[itemIdx]
	player.Warehouse = append(player.Warehouse[:itemIdx], player.Warehouse[itemIdx+1:]...)
	guild.Warehouse = append(guild.Warehouse, item)

	itemTypeName := item.TypeID
	if itemType, ok := models.GetItemType(item.TypeID); ok {
		itemTypeName = itemType.Name
	}
	rm.addGuildLog(room, guild, playerID, player.Name, fmt.Sprintf("存入了%s", itemTypeName))

	return ""
}

func (rm *RoomManager) WithdrawGuildWarehouse(roomID, playerID, itemID string) string {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	room, ok := rm.rooms[roomID]
	if !ok {
		return "房间不存在"
	}

	player := room.Players[playerID]
	if player == nil || player.GuildID == "" {
		return "你没有加入任何公会"
	}

	guild, ok := room.Guilds[player.GuildID]
	if !ok {
		return "公会不存在"
	}

	if len(player.Warehouse) >= player.WarehouseCapacity {
		return "你的个人仓库已满"
	}

	itemIdx := -1
	for i, item := range guild.Warehouse {
		if item.ID == itemID {
			itemIdx = i
			break
		}
	}
	if itemIdx == -1 {
		return "公会仓库中未找到该物品"
	}

	item := guild.Warehouse[itemIdx]
	guild.Warehouse = append(guild.Warehouse[:itemIdx], guild.Warehouse[itemIdx+1:]...)
	player.Warehouse = append(player.Warehouse, item)

	itemTypeName := item.TypeID
	if itemType, ok := models.GetItemType(item.TypeID); ok {
		itemTypeName = itemType.Name
	}
	rm.addGuildLog(room, guild, playerID, player.Name, fmt.Sprintf("取出了%s", itemTypeName))

	return ""
}

func (rm *RoomManager) distributeGuildWarehouseOnDisband(room *models.Room, guild *models.Guild, disbanderID string) {
	if len(guild.Warehouse) == 0 {
		return
	}

	onlineMembers := make([]*models.PlayerState, 0)
	for _, member := range guild.Members {
		if player, ok := room.Players[member.PlayerID]; ok {
			onlineMembers = append(onlineMembers, player)
		}
	}

	if len(onlineMembers) == 0 {
		return
	}

	sort.SliceStable(onlineMembers, func(i, j int) bool {
		return getMemberJoinTime(guild, onlineMembers[i].ID) < getMemberJoinTime(guild, onlineMembers[j].ID)
	})

	items := make([]models.Item, len(guild.Warehouse))
	copy(items, guild.Warehouse)
	guild.Warehouse = guild.Warehouse[:0]

	memberIdx := 0
	for len(items) > 0 {
		player := onlineMembers[memberIdx%len(onlineMembers)]
		if len(player.Warehouse) < player.WarehouseCapacity {
			player.Warehouse = append(player.Warehouse, items[0])
			items = items[1:]
			memberIdx++
			continue
		}

		allFull := true
		for _, p := range onlineMembers {
			if len(p.Warehouse) < p.WarehouseCapacity {
				allFull = false
				break
			}
		}
		if allFull {
			break
		}
		memberIdx++
	}

	if len(items) > 0 {
		disbander := room.Players[disbanderID]
		if disbander != nil {
			disbander.Warehouse = append(disbander.Warehouse, items...)
		}
	}
}

func getMemberJoinTime(guild *models.Guild, playerID string) int64 {
	for _, m := range guild.Members {
		if m.PlayerID == playerID {
			return m.JoinTime
		}
	}
	return 0
}

func (rm *RoomManager) addGuildLog(room *models.Room, guild *models.Guild, playerID, playerName, action string) {
	if guild == nil {
		return
	}

	logEntry := models.GuildLogEntry{
		ID:         models.NewID(),
		Timestamp:  time.Now().Unix(),
		PlayerID:   playerID,
		PlayerName: playerName,
		Action:     action,
	}

	guild.Logs = append([]models.GuildLogEntry{logEntry}, guild.Logs...)

	if len(guild.Logs) > models.GuildMaxLogs {
		guild.Logs = guild.Logs[:models.GuildMaxLogs]
	}
}

func (rm *RoomManager) generateGuildTasks(room *models.Room, guild *models.Guild) {
	if guild == nil || room == nil {
		return
	}

	if guild.LastTaskWeek >= room.CurrentWeek && len(guild.Tasks) > 0 {
		return
	}

	guild.Tasks = make([]models.GuildTask, 0)
	taskTypes := []models.GuildTaskType{
		models.GuildTaskTypeSellItems,
		models.GuildTaskTypeTreasuryGold,
		models.GuildTaskTypeMemberCount,
	}

	shuffleIdx := rand.Perm(len(taskTypes))
	selectedTypes := make([]models.GuildTaskType, 0, models.GuildTasksPerWeek)
	for i := 0; i < models.GuildTasksPerWeek && i < len(taskTypes); i++ {
		selectedTypes = append(selectedTypes, taskTypes[shuffleIdx[i]])
	}

	for _, taskType := range selectedTypes {
		baseTarget := models.GuildTaskBaseTargets[taskType]
		target := baseTarget * guild.Level
		if taskType == models.GuildTaskTypeMemberCount {
			baseMax := getGuildMaxMembers(room.MaxPlayers)
			target = guild.Level + 1
			if target > baseMax {
				target = baseMax
			}
		}

		baseReward := models.GuildTaskBaseRewards[taskType]
		description := fmt.Sprintf(models.GuildTaskDescriptions[taskType], target)

		task := models.GuildTask{
			ID:          models.NewID(),
			Type:        taskType,
			Target:      target,
			Progress:    0,
			Completed:   false,
			RewardGold:  baseReward.Gold * guild.Level,
			RewardExp:   baseReward.Exp * guild.Level,
			Description: description,
		}

		if taskType == models.GuildTaskTypeMemberCount {
			task.Progress = len(guild.Members)
			if task.Progress >= task.Target {
				task.Completed = true
			}
		}

		guild.Tasks = append(guild.Tasks, task)
	}

	guild.LastTaskWeek = room.CurrentWeek
}

func (rm *RoomManager) RefreshGuildTasksForAllGuilds(roomID string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	room, ok := rm.rooms[roomID]
	if !ok {
		return
	}

	for _, guild := range room.Guilds {
		rm.generateGuildTasks(room, guild)
	}
}

func (rm *RoomManager) UpdateGuildTaskProgress(roomID, guildID string, taskType models.GuildTaskType, amount int) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	room, ok := rm.rooms[roomID]
	if !ok {
		return
	}

	guild, ok := room.Guilds[guildID]
	if !ok {
		return
	}

	rm.updateGuildTaskProgressLocked(room, guild, taskType, amount)
}

func (rm *RoomManager) updateGuildTaskProgressLocked(room *models.Room, guild *models.Guild, taskType models.GuildTaskType, amount int) {
	if room == nil || guild == nil {
		return
	}

	rm.generateGuildTasksLocked(room, guild)

	for i := range guild.Tasks {
		if guild.Tasks[i].Type != taskType || guild.Tasks[i].Completed {
			continue
		}

		if taskType == models.GuildTaskTypeMemberCount {
			guild.Tasks[i].Progress = len(guild.Members)
		} else {
			guild.Tasks[i].Progress += amount
		}

		if guild.Tasks[i].Progress >= guild.Tasks[i].Target {
			guild.Tasks[i].Progress = guild.Tasks[i].Target
			guild.Tasks[i].Completed = true

			rewardGold := guild.Tasks[i].RewardGold
			rewardExp := guild.Tasks[i].RewardExp
			taskDesc := guild.Tasks[i].Description

			for _, member := range guild.Members {
				player, ok := room.Players[member.PlayerID]
				if ok {
					player.Gold += rewardGold
					for j := range guild.Members {
						if guild.Members[j].PlayerID == member.PlayerID {
							guild.Members[j].TotalExp += rewardExp
							break
						}
					}
				}
			}

			rm.addGuildLogLocked(room, guild, "", "系统", fmt.Sprintf("任务完成：%s，奖励%d金币%d经验", taskDesc, rewardGold, rewardExp))
		}
	}
}

func (rm *RoomManager) CheckAndCompleteMemberCountTask(roomID, guildID string) {
	rm.UpdateGuildTaskProgress(roomID, guildID, models.GuildTaskTypeMemberCount, 0)
}

func (rm *RoomManager) TransferGuildLeadership(roomID, currentLeaderID, newLeaderID string) string {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	room, ok := rm.rooms[roomID]
	if !ok {
		return "房间不存在"
	}

	player := room.Players[currentLeaderID]
	if player == nil || player.GuildID == "" {
		return "你没有加入任何公会"
	}

	guild, ok := room.Guilds[player.GuildID]
	if !ok {
		return "公会不存在"
	}

	if guild.LeaderID != currentLeaderID {
		return "只有会长可以转让"
	}

	if currentLeaderID == newLeaderID {
		return "不能转让给自己"
	}

	newLeaderName := ""
	found := false
	for i := range guild.Members {
		if guild.Members[i].PlayerID == newLeaderID {
			guild.Members[i].IsLeader = true
			newLeaderName = guild.Members[i].PlayerName
			found = true
		} else if guild.Members[i].PlayerID == currentLeaderID {
			guild.Members[i].IsLeader = false
		}
	}

	if !found {
		return "目标玩家不在公会中"
	}

	guild.LeaderID = newLeaderID

	rm.addGuildLog(room, guild, currentLeaderID, player.Name, fmt.Sprintf("将会长转让给%s", newLeaderName))

	return ""
}

func (rm *RoomManager) CalculateGuildMemberRanking(room *models.Room, guild *models.Guild) []models.GuildMemberRank {
	if guild == nil {
		return nil
	}

	ranks := make([]models.GuildMemberRank, 0, len(guild.Members))
	var leaderRank *models.GuildMemberRank

	for _, member := range guild.Members {
		joinWeeks := 0
		if room != nil {
			joinWeeks = room.CurrentWeek - member.JoinWeek
			if joinWeeks < 0 {
				joinWeeks = 0
			}
		}

		rank := models.GuildMemberRank{
			PlayerID:     member.PlayerID,
			PlayerName:   member.PlayerName,
			Contribution: member.Contribution,
			JoinWeeks:    joinWeeks,
			IsLeader:     member.IsLeader,
		}

		if member.IsLeader {
			leaderRank = &rank
		} else {
			ranks = append(ranks, rank)
		}
	}

	sort.SliceStable(ranks, func(i, j int) bool {
		return ranks[i].Contribution > ranks[j].Contribution
	})

	for i := range ranks {
		ranks[i].Rank = i + 1
	}

	result := make([]models.GuildMemberRank, 0, len(guild.Members))
	if leaderRank != nil {
		leaderRank.Rank = 0
		result = append(result, *leaderRank)
	}
	result = append(result, ranks...)

	return result
}

func (rm *RoomManager) GetGuildMemberRanking(roomID, guildID string) []models.GuildMemberRank {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	room, ok := rm.rooms[roomID]
	if !ok {
		return nil
	}

	guild, ok := room.Guilds[guildID]
	if !ok {
		return nil
	}

	return rm.CalculateGuildMemberRanking(room, guild)
}
