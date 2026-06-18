package game

import (
	"math"
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
	}

	guild.Members = append(guild.Members, models.GuildMember{
		PlayerID:     playerID,
		PlayerName:   player.Name,
		JoinTime:     time.Now().Unix(),
		IsLeader:     true,
		Contribution: 0,
	})

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

	if banExpireTime, banned := guild.BannedPlayers[playerID]; banned {
		banDurationWeeks := int64(models.GuildKickBanDuration)
		weeksSinceBan := (room.CurrentWeek - int(banExpireTime/int64(models.GuildKickBanDuration)))
		if weeksSinceBan < banDurationWeeks {
			return nil, "你被该公会踢出未满2周，暂不能加入"
		}
		delete(guild.BannedPlayers, playerID)
	}

	guild.Members = append(guild.Members, models.GuildMember{
		PlayerID:     playerID,
		PlayerName:   player.Name,
		JoinTime:     time.Now().Unix(),
		IsLeader:     false,
		Contribution: 0,
	})

	player.GuildID = guildID

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

	if len(guild.Members) == 1 {
		rm.distributeGuildWarehouseOnDisband(room, guild)
		delete(room.Guilds, guild.ID)
		player.GuildID = ""
		return ""
	}

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

	memberIdx := -1
	for i, member := range guild.Members {
		if member.PlayerID == targetPlayerID {
			memberIdx = i
			break
		}
	}
	if memberIdx >= 0 {
		guild.Members = append(guild.Members[:memberIdx], guild.Members[memberIdx+1:]...)
	}

	guild.BannedPlayers[targetPlayerID] = int64(room.CurrentWeek)
	targetPlayer.GuildID = ""

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

	return ""
}

func (rm *RoomManager) distributeGuildWarehouseOnDisband(room *models.Room, guild *models.Guild) {
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

	sortedMembers := make([]*models.PlayerState, len(onlineMembers))
	copy(sortedMembers, onlineMembers)
	for i := range sortedMembers {
		for j := i + 1; j < len(sortedMembers); j++ {
			var joinTimeI, joinTimeJ int64
			for _, m := range guild.Members {
				if m.PlayerID == sortedMembers[i].ID {
					joinTimeI = m.JoinTime
				}
				if m.PlayerID == sortedMembers[j].ID {
					joinTimeJ = m.JoinTime
				}
			}
			if joinTimeI > joinTimeJ {
				sortedMembers[i], sortedMembers[j] = sortedMembers[j], sortedMembers[i]
			}
		}
	}

	warehouseItems := make([]models.Item, len(guild.Warehouse))
	copy(warehouseItems, guild.Warehouse)

	memberIdx := 0
	for len(warehouseItems) > 0 {
		if len(sortedMembers) == 0 {
			break
		}

		player := sortedMembers[memberIdx%len(sortedMembers)]
		if len(player.Warehouse) < player.WarehouseCapacity {
			player.Warehouse = append(player.Warehouse, warehouseItems[0])
			warehouseItems = warehouseItems[1:]
		}
		memberIdx++

		if memberIdx > len(sortedMembers)*2 {
			break
		}
	}
}
