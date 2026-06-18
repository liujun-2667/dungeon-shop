package game

import (
	"sync"
	"time"

	"dungeon-shop/internal/models"
)

type RoomManager struct {
	rooms map[string]*models.Room
	mu    sync.RWMutex
}

func NewRoomManager() *RoomManager {
	return &RoomManager{
		rooms: make(map[string]*models.Room),
	}
}

func (rm *RoomManager) CreateRoom(name string, maxPlayers int, seed int64) *models.Room {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	room := models.NewRoom(name, maxPlayers, seed)
	rm.rooms[room.ID] = room
	return room
}

func (rm *RoomManager) GetRoom(roomID string) (*models.Room, bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	room, ok := rm.rooms[roomID]
	return room, ok
}

func (rm *RoomManager) ListRooms() []*models.Room {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	rooms := make([]*models.Room, 0, len(rm.rooms))
	for _, room := range rm.rooms {
		rooms = append(rooms, room)
	}
	return rooms
}

func (rm *RoomManager) JoinRoom(roomID, playerName, shopName string) (*models.Room, *models.PlayerState, bool) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	room, ok := rm.rooms[roomID]
	if !ok {
		return nil, nil, false
	}

	if len(room.Players) >= room.MaxPlayers {
		return room, nil, false
	}

	if room.Status != "waiting" {
		return room, nil, false
	}

	player := models.NewPlayerState(playerName, shopName)

	playerIdx := len(room.Players)
	recipes := GenerateRecipes(room.Seed+int64(playerIdx)*100, 2)
	for i := range recipes {
		recipes[i].OwnerID = player.ID
	}
	player.Recipes = recipes

	room.Players[player.ID] = player
	return room, player, true
}

func (rm *RoomManager) StartGame(roomID string) bool {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	room, ok := rm.rooms[roomID]
	if !ok {
		return false
	}

	if len(room.Players) < 2 {
		return false
	}

	room.Status = "playing"
	room.CurrentWeek = 1
	room.Phase = models.PhasePurchase
	room.PhaseEndTime = time.Now().Unix() + int64(room.PhaseDuration)

	room.WholesalerStock = GenerateWholesalerStock(room.Seed, room.CurrentWeek)
	playerIDs := make([]string, 0, len(room.Players))
	for id := range room.Players {
		playerIDs = append(playerIDs, id)
	}
	room.NPCsThisWeek = GenerateNPCs(room.Seed, room.CurrentWeek, len(room.Players), playerIDs, room.CurrentEvent)

	return true
}

func (rm *RoomManager) ProcessPhaseEnd(roomID string) bool {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	room, ok := rm.rooms[roomID]
	if !ok || room.Status != "playing" {
		return false
	}

	switch room.Phase {
	case models.PhasePurchase:
		room.Phase = models.PhaseBusiness
		room.NPCIndex = 0
		room.PendingBargain = nil
		room.BusinessLogs = make([]models.BusinessLogEntry, 0)
		room.NPCsThisWeek = ShuffleNPCs(room.NPCsThisWeek, room.Seed, room.CurrentWeek)
	case models.PhaseBusiness:
		room.Phase = models.PhaseExplore
		room.PendingBargain = nil
		ProcessExplorePhase(room)
	case models.PhaseExplore:
		room.CurrentWeek++
		if room.CurrentWeek > room.TotalWeeks {
			room.Status = "finished"
			room.Phase = models.PhaseSettlement
			return true
		}

		rm.settleAuctionsLocked(room)

		for _, player := range room.Players {
			expiredCount := ProcessExpiredItems(player, room.CurrentWeek)
			if expiredCount > 0 {
				player.Reputation -= expiredCount * 2
			}
			CheckBankruptcy(player)
			player.AssetHistory = append(player.AssetHistory, CalculateTotalAssets(player))
			player.WeeklyStats = models.WeeklyStats{}
		}

		room.WholesalerStock = GenerateWholesalerStock(room.Seed, room.CurrentWeek)

		if room.CurrentEvent == nil || room.CurrentWeek > room.CurrentEvent.StartWeek+room.CurrentEvent.Duration-1 {
			newEvent := GenerateRandomEvent(room.Seed, room.CurrentWeek)
			room.CurrentEvent = newEvent
			if newEvent != nil && newEvent.Effects.StealItem {
				for _, player := range room.Players {
					if !player.IsBankrupt && len(player.Warehouse) > 0 {
						sr := NewSeededRand(room.Seed + int64(room.CurrentWeek)*1000)
						idx := sr.Intn(len(player.Warehouse))
						player.Warehouse = append(player.Warehouse[:idx], player.Warehouse[idx+1:]...)
					}
				}
			}
		}

		npcPlayerIDs := make([]string, 0, len(room.Players))
		for id := range room.Players {
			npcPlayerIDs = append(npcPlayerIDs, id)
		}
		room.NPCsThisWeek = GenerateNPCs(room.Seed, room.CurrentWeek, len(room.Players), npcPlayerIDs, room.CurrentEvent)

		room.Phase = models.PhasePurchase
	}

	room.PhaseEndTime = time.Now().Unix() + int64(room.PhaseDuration)
	return true
}

func (rm *RoomManager) ProcessNextNPC(roomID string) (*models.BargainRequest, string, []models.BusinessLogEntry, bool) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	room, ok := rm.rooms[roomID]
	if !ok || room.Status != "playing" || room.Phase != models.PhaseBusiness {
		return nil, "", nil, false
	}

	if room.PendingBargain != nil {
		return nil, "", nil, true
	}

	if room.NPCIndex >= len(room.NPCsThisWeek) {
		return nil, "", nil, false
	}

	npcIdx := room.NPCIndex
	npc := room.NPCsThisWeek[npcIdx]
	room.NPCIndex++

	npcSeed := room.Seed + int64(room.CurrentWeek)*6000 + int64(npcIdx)*100
	npcRand := NewSeededRand(npcSeed)

	playerIDs := make([]string, 0, len(room.Players))
	for id := range room.Players {
		playerIDs = append(playerIDs, id)
	}

	playerID := ""
	if npc.IsVIP {
		if npc.TargetPlayerID != "" {
			if p, ok := room.Players[npc.TargetPlayerID]; ok && !p.IsBankrupt {
				playerID = npc.TargetPlayerID
			}
		}
		if playerID == "" {
			nonBankrupt := make([]string, 0)
			for _, id := range playerIDs {
				if !room.Players[id].IsBankrupt {
					nonBankrupt = append(nonBankrupt, id)
				}
			}
			if len(nonBankrupt) > 0 {
				targetIdx := npcRand.Intn(len(nonBankrupt))
				playerID = nonBankrupt[targetIdx]
			}
		}
	} else {
		visitOrder := npcRand.Perm(len(playerIDs))
		shopsVisited := 0

		for _, idx := range visitOrder {
			if shopsVisited >= npc.MaxShopsToVisit {
				break
			}

			candidateID := playerIDs[idx]
			candidate := room.Players[candidateID]

			if candidate.IsBankrupt {
				continue
			}

			visitRoll := npcRand.Float64()
			adjustedBonus := candidate.AttractionBonus
			if shopsVisited == 0 {
				adjustedBonus = 1.0
			}
			repLevel := models.GetReputationLevel(candidate.Reputation)
			visitBonus := models.GetReputationVisitBonus(repLevel)
			finalThreshold := adjustedBonus + visitBonus
			if finalThreshold < 0 {
				finalThreshold = 0
			}
			if visitRoll > finalThreshold {
				continue
			}

			shopsVisited++
			playerID = candidateID
			break
		}
	}

	if playerID == "" {
		return nil, "", rm.popLogs(room), room.NPCIndex < len(room.NPCsThisWeek)
	}

	bargain, logs, hasMore := rm.processSingleNPCShopping(room, npc, playerID, npcIdx)
	if logs != nil {
		room.BusinessLogs = append(room.BusinessLogs, logs...)
	}
	return bargain, playerID, rm.popLogs(room), hasMore || room.NPCIndex < len(room.NPCsThisWeek)
}

func (rm *RoomManager) popLogs(room *models.Room) []models.BusinessLogEntry {
	if len(room.BusinessLogs) == 0 {
		return nil
	}
	logs := room.BusinessLogs
	room.BusinessLogs = make([]models.BusinessLogEntry, 0)
	return logs
}

func (rm *RoomManager) processSingleNPCShopping(room *models.Room, npc models.NPC, playerID string, npcIdx int) (*models.BargainRequest, []models.BusinessLogEntry, bool) {
	player := room.Players[playerID]
	if player == nil || player.IsBankrupt {
		return nil, nil, false
	}

	npcSeed := room.Seed + int64(room.CurrentWeek)*7000 + int64(npcIdx)*50
	npcRand := NewSeededRand(npcSeed)

	repLevel := models.GetReputationLevel(player.Reputation)
	budgetBonus := models.GetReputationBudgetBonus(repLevel)
	budget := int(float64(npc.Budget) * (1.0 + budgetBonus))

	logs := make([]models.BusinessLogEntry, 0)

	allSlots := make([]*models.ShelfSlot, 0)
	for i := range player.Shelves {
		allSlots = append(allSlots, &player.Shelves[i])
	}

	for _, branch := range player.BranchShops {
		for i := range branch.Shelves {
			allSlots = append(allSlots, &branch.Shelves[i])
		}
	}

	validSlots := make([]*models.ShelfSlot, 0)
	for _, slot := range allSlots {
		if slot.Item != nil && slot.Price > 0 && slot.Price <= budget {
			if _, ok := models.GetItemType(slot.Item.TypeID); ok {
				validSlots = append(validSlots, slot)
			}
		}
	}

	type scoredSlot struct {
		slot  *models.ShelfSlot
		score float64
	}
	scored := make([]scoredSlot, 0, len(validSlots))
	for _, slot := range validSlots {
		itemType, _ := models.GetItemType(slot.Item.TypeID)
		pref := npc.Preferences[itemType.Category]
		qualityMult := 1.0 + (float64(slot.Item.QualityRank()) * npc.QualityPreference * 0.3)
		impulseBonus := (0.7 + npc.Impulsiveness * 0.6)
		randomFactor := 0.8 + npcRand.Float64()*0.4
		score := pref * qualityMult * impulseBonus * randomFactor
		scored = append(scored, scoredSlot{slot: slot, score: score})
	}

	for i := len(scored) - 1; i > 0; i-- {
		j := npcRand.Intn(i + 1)
		scored[i], scored[j] = scored[j], scored[i]
	}

	for i := 0; i < len(scored)-1; i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].score > scored[i].score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	for _, ss := range scored {
		slot := ss.slot
		itemType, _ := models.GetItemType(slot.Item.TypeID)

		prob := CalculatePurchaseProbability(npc, itemType, slot.Price, room.CurrentEvent)
		prob *= (0.8 + npc.Impulsiveness*0.4)

		if npc.IsVIP {
			prob *= 1.5
		}

		prices := GetAllPlayerPrices(room, slot.Item.TypeID)
		if len(prices) > 1 {
			minPrice := slot.Price
			for _, p := range prices {
				if p < minPrice {
					minPrice = p
				}
			}
			if slot.Price > minPrice {
				priceGap := float64(slot.Price-minPrice) / float64(minPrice)
				sensitivityMult := 1.0 - priceGap*npc.PriceSensitivity
				if sensitivityMult < 0.1 {
					sensitivityMult = 0.1
				}
				prob *= sensitivityMult
			} else if slot.Price == minPrice {
				prob *= 1.3
			}
		}

		if prob > 1.0 {
			prob = 1.0
		}
		if prob < 0 {
			prob = 0
		}

		if npcRand.Float64() < prob {
			basePrice := itemType.BasePrice

			if npcRand.Float64() < 0.2 {
				discountPct := 0.1 + npcRand.Float64()*0.15
				bargainedPrice := int(float64(slot.Price) * (1.0 - discountPct))
				if bargainedPrice < 1 {
					bargainedPrice = 1
				}

				bargainID := models.NewID()
				bargain := &models.BargainRequest{
					ID:             bargainID,
					NPCID:          npc.ID,
					NPCName:        npc.Name,
					NPCClass:       npc.Class,
					ShelfID:        slot.ID,
					ItemTypeID:     slot.Item.TypeID,
					ItemName:       itemType.Name,
					ItemQuality:    slot.Item.Quality,
					OriginalPrice:  slot.Price,
					BargainedPrice: bargainedPrice,
					ExpiresAt:      time.Now().Unix() + 5,
				}

				room.PendingBargain = bargain
				room.BargainNPCIdx = npcIdx
				room.BargainSlot = slot
				npcCopy := npc
				room.BargainNPC = &npcCopy
				room.BargainPlayerID = playerID

				logs = append(logs, models.BusinessLogEntry{
					PlayerID: playerID,
					NPCName:  npc.Name,
					Message:  "发起了砍价请求",
					Type:     "bargain_start",
				})

				return bargain, logs, true
			}

			priceTier := GetPriceTier(slot.Price, basePrice)
			switch priceTier {
			case PriceTierBargain:
				player.Reputation++
			case PriceTierOverpriced:
				player.Reputation--
			}

			player.Gold += slot.Price
			player.WeeklyStats.Income += slot.Price
			player.WeeklyStats.ItemsSold++

			logs = append(logs, models.BusinessLogEntry{
				PlayerID: playerID,
				NPCName:  npc.Name,
				Message:  "购买了 " + itemType.Name + " (" + models.GetQualityName(slot.Item.Quality) + ")",
				Type:     "purchase",
			})

			slot.Item = nil
			slot.ItemID = ""
			slot.Price = 0

			return nil, logs, true
		}
	}

	logs = append(logs, models.BusinessLogEntry{
		PlayerID: playerID,
		NPCName:  npc.Name,
		Message:  "离开了店铺",
		Type:     "leave",
	})

	return nil, logs, true
}

func (rm *RoomManager) ResolveBargain(roomID, bargainID string, accepted bool) []models.BusinessLogEntry {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	room, ok := rm.rooms[roomID]
	if !ok || room.PendingBargain == nil {
		return nil
	}

	if room.PendingBargain.ID != bargainID {
		return nil
	}

	bargain := room.PendingBargain
	slot := room.BargainSlot
	npc := room.BargainNPC
	playerID := room.BargainPlayerID
	player := room.Players[playerID]

	logs := make([]models.BusinessLogEntry, 0)
	_ = bargain

	if slot.Item == nil {
		room.PendingBargain = nil
		room.BargainSlot = nil
		room.BargainNPC = nil
		room.BargainPlayerID = ""
		return logs
	}

	itemType, _ := models.GetItemType(slot.Item.TypeID)
	basePrice := itemType.BasePrice
	npcRand := NewSeededRand(room.Seed + int64(room.CurrentWeek)*9000 + int64(room.BargainNPCIdx)*33)

	if accepted {
		player.Reputation++

		player.Gold += bargain.BargainedPrice
		player.WeeklyStats.Income += bargain.BargainedPrice
		player.WeeklyStats.ItemsSold++

		logs = append(logs, models.BusinessLogEntry{
			PlayerID: playerID,
			NPCName:  npc.Name,
			Message:  "砍价成功，购买了 " + itemType.Name,
			Type:     "bargain_success",
		})

		slot.Item = nil
		slot.ItemID = ""
		slot.Price = 0
	} else {
		if npcRand.Float64() < 0.5 {
			player.Reputation--

			logs = append(logs, models.BusinessLogEntry{
				PlayerID: playerID,
				NPCName:  npc.Name,
				Message:  "砍价被拒绝，愤怒离开",
				Type:     "bargain_reject_leave",
			})
		} else {
			priceTier := GetPriceTier(bargain.OriginalPrice, basePrice)
			switch priceTier {
			case PriceTierBargain:
				player.Reputation++
			case PriceTierOverpriced:
				player.Reputation--
			}

			player.Gold += bargain.OriginalPrice
			player.WeeklyStats.Income += bargain.OriginalPrice
			player.WeeklyStats.ItemsSold++

			logs = append(logs, models.BusinessLogEntry{
				PlayerID: playerID,
				NPCName:  npc.Name,
				Message:  "接受原价，购买了 " + itemType.Name,
				Type:     "bargain_reject_buy",
			})

			slot.Item = nil
			slot.ItemID = ""
			slot.Price = 0
		}
	}

	room.PendingBargain = nil
	room.BargainSlot = nil
	room.BargainNPC = nil
	room.BargainPlayerID = ""

	room.BusinessLogs = append(room.BusinessLogs, logs...)
	return rm.popLogs(room)
}

func (rm *RoomManager) HasPendingBargain(roomID string) (*models.BargainRequest, string) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	room, ok := rm.rooms[roomID]
	if !ok || room.PendingBargain == nil {
		return nil, ""
	}
	return room.PendingBargain, room.BargainPlayerID
}

func ProcessExplorePhase(room *models.Room) {
	sr := NewSeededRand(room.Seed + int64(room.CurrentWeek)*8000)

	if room.CurrentEvent != nil && room.CurrentEvent.Effects.BlockExploration {
		for _, mission := range room.ExplorationMissions {
			player := room.Players[mission.PlayerID]
			if player == nil {
				continue
			}
			for i := range player.Adventurers {
				if player.Adventurers[i].ID == mission.AdventurerID {
					player.Adventurers[i].IsOnMission = false
					player.Adventurers[i].IsInjured = true
					player.Adventurers[i].InjuredUntilWeek = room.CurrentWeek + 2
					break
				}
			}
		}
		room.ExplorationMissions = make([]models.ExplorationMission, 0)
		return
	}

	newTasks := make([]models.SynthesisTask, 0)
	for _, task := range room.SynthesisTasks {
		if task.CompleteWeek <= room.CurrentWeek {
			player := room.Players[task.PlayerID]
			if player == nil || player.IsBankrupt {
				continue
			}

			var recipe *models.Recipe
			for i := range player.Recipes {
				if player.Recipes[i].ID == task.RecipeID {
					recipe = &player.Recipes[i]
					break
				}
			}

			if recipe != nil && len(player.Warehouse) < player.WarehouseCapacity {
				itemType := models.ItemTypes[recipe.OutputItemType]
				newItem := models.Item{
					ID:           models.NewID(),
					TypeID:       recipe.OutputItemType,
					Quality:      recipe.OutputQuality,
					PurchaseCost: int(float64(itemType.BasePrice) * models.QualityMultiplier[recipe.OutputQuality] * 0.6),
				}
				player.Warehouse = append(player.Warehouse, newItem)
			}
		} else {
			newTasks = append(newTasks, task)
		}
	}
	room.SynthesisTasks = newTasks

	completedMissions := make([]models.ExplorationMission, 0)
	for _, mission := range room.ExplorationMissions {
		player := room.Players[mission.PlayerID]
		if player == nil || player.IsBankrupt {
			completedMissions = append(completedMissions, mission)
			continue
		}

		var adv *models.Adventurer
		for i := range player.Adventurers {
			if player.Adventurers[i].ID == mission.AdventurerID {
				adv = &player.Adventurers[i]
				break
			}
		}

		if adv == nil {
			completedMissions = append(completedMissions, mission)
			continue
		}

		successRate := CalculateExplorationSuccessRate(adv.Level, mission.Floor)

		if sr.Float64() < successRate {
			loot := GetFloorLoot(mission.Floor, room.Seed, room.CurrentWeek)
			for _, item := range loot {
				if len(player.Warehouse) < player.WarehouseCapacity {
					player.Warehouse = append(player.Warehouse, item)
				}
			}
			adv.IsOnMission = false
		} else {
			adv.IsOnMission = false
			adv.IsInjured = true
			adv.InjuredUntilWeek = room.CurrentWeek + 2
		}

		completedMissions = append(completedMissions, mission)
	}

	remaining := make([]models.ExplorationMission, 0)
	for _, m := range room.ExplorationMissions {
		found := false
		for _, cm := range completedMissions {
			if m.AdventurerID == cm.AdventurerID && m.PlayerID == cm.PlayerID {
				found = true
				break
			}
		}
		if !found {
			remaining = append(remaining, m)
		}
	}
	room.ExplorationMissions = remaining
}

func (rm *RoomManager) BuyFromWholesaler(roomID, playerID, itemTypeID string, quality models.Quality, quantity int) bool {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	room, ok := rm.rooms[roomID]
	if !ok || room.Phase != models.PhasePurchase {
		return false
	}

	player := room.Players[playerID]
	if player == nil || player.IsBankrupt {
		return false
	}

	stockIdx := -1
	for i, item := range room.WholesalerStock {
		if item.TypeID == itemTypeID && item.Quality == quality && item.Quantity >= quantity {
			stockIdx = i
			break
		}
	}

	if stockIdx == -1 {
		return false
	}

	totalCost := room.WholesalerStock[stockIdx].Price * quantity
	if player.Gold < totalCost {
		return false
	}

	if len(player.Warehouse)+quantity > player.WarehouseCapacity {
		return false
	}

	itemType := models.ItemTypes[itemTypeID]
	for i := 0; i < quantity; i++ {
		item := models.Item{
			ID:           models.NewID(),
			TypeID:       itemTypeID,
			Quality:      quality,
			PurchaseCost: room.WholesalerStock[stockIdx].Price,
		}
		if itemType.HasShelfLife {
			item.ExpiresWeek = room.CurrentWeek + itemType.ShelfLifeWeeks
		}
		player.Warehouse = append(player.Warehouse, item)
	}

	player.Gold -= totalCost
	player.WeeklyStats.Expense += totalCost
	player.WeeklyStats.ItemsBought += quantity

	room.WholesalerStock[stockIdx].Quantity -= quantity
	if room.WholesalerStock[stockIdx].Quantity <= 0 {
		room.WholesalerStock = append(room.WholesalerStock[:stockIdx], room.WholesalerStock[stockIdx+1:]...)
	}

	return true
}

func (rm *RoomManager) PlaceItemOnShelf(roomID, playerID, itemID, shelfID string, price int) bool {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	room, ok := rm.rooms[roomID]
	if !ok || room.Phase != models.PhasePurchase {
		return false
	}

	player := room.Players[playerID]
	if player == nil || player.IsBankrupt {
		return false
	}

	itemIdx := -1
	for i, item := range player.Warehouse {
		if item.ID == itemID {
			itemIdx = i
			break
		}
	}

	if itemIdx == -1 {
		return false
	}

	shelfIdx := -1
	for i, slot := range player.Shelves {
		if slot.ID == shelfID {
			shelfIdx = i
			break
		}
	}

	if shelfIdx == -1 {
		return false
	}

	item := player.Warehouse[itemIdx]

	if player.Shelves[shelfIdx].Item != nil {
		player.Warehouse = append(player.Warehouse, *player.Shelves[shelfIdx].Item)
	}

	player.Shelves[shelfIdx].Item = &item
	player.Shelves[shelfIdx].ItemID = item.ID
	player.Shelves[shelfIdx].Price = price

	player.Warehouse = append(player.Warehouse[:itemIdx], player.Warehouse[itemIdx+1:]...)

	return true
}

func (rm *RoomManager) SetShelfPrice(roomID, playerID, shelfID string, price int) bool {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	room, ok := rm.rooms[roomID]
	if !ok || room.Phase != models.PhasePurchase {
		return false
	}

	player := room.Players[playerID]
	if player == nil || player.IsBankrupt {
		return false
	}

	for i := range player.Shelves {
		if player.Shelves[i].ID == shelfID {
			player.Shelves[i].Price = price
			return true
		}
	}

	return false
}

func (rm *RoomManager) HireAdventurer(roomID, playerID string, adventurerIdx int) (models.Adventurer, bool) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	room, ok := rm.rooms[roomID]
	if !ok || room.Phase != models.PhasePurchase {
		return models.Adventurer{}, false
	}

	player := room.Players[playerID]
	if player == nil || player.IsBankrupt {
		return models.Adventurer{}, false
	}

	if len(player.Adventurers) >= player.MaxAdventurers {
		return models.Adventurer{}, false
	}

	adv := GenerateAdventurer(room.Seed+int64(room.CurrentWeek)*1000, adventurerIdx)

	if player.Gold < adv.HireCost {
		return models.Adventurer{}, false
	}

	player.Gold -= adv.HireCost
	player.WeeklyStats.Expense += adv.HireCost
	player.Adventurers = append(player.Adventurers, adv)

	return adv, true
}

func (rm *RoomManager) DispatchAdventurer(roomID, playerID, adventurerID string, floor int) bool {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	room, ok := rm.rooms[roomID]
	if !ok || room.Phase != models.PhaseExplore {
		return false
	}

	player := room.Players[playerID]
	if player == nil || player.IsBankrupt {
		return false
	}

	if floor < 1 || floor > 5 {
		return false
	}

	advIdx := -1
	for i, adv := range player.Adventurers {
		if adv.ID == adventurerID {
			advIdx = i
			break
		}
	}

	if advIdx == -1 {
		return false
	}

	adv := &player.Adventurers[advIdx]
	if adv.IsOnMission || adv.IsInjured {
		return false
	}

	adv.IsOnMission = true

	room.ExplorationMissions = append(room.ExplorationMissions, models.ExplorationMission{
		PlayerID:     playerID,
		AdventurerID: adventurerID,
		Floor:        floor,
		Week:         room.CurrentWeek,
	})

	return true
}

func (rm *RoomManager) StartSynthesis(roomID, playerID, recipeID string) bool {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	room, ok := rm.rooms[roomID]
	if !ok || room.Phase != models.PhasePurchase {
		return false
	}

	player := room.Players[playerID]
	if player == nil || player.IsBankrupt {
		return false
	}

	var recipe *models.Recipe
	for i := range player.Recipes {
		if player.Recipes[i].ID == recipeID {
			recipe = &player.Recipes[i]
			break
		}
	}

	if recipe == nil {
		return false
	}

	matCounts := make(map[string]int)
	for _, mat := range recipe.Materials {
		matCounts[mat]++
	}

	warehouseCopy := make([]models.Item, len(player.Warehouse))
	copy(warehouseCopy, player.Warehouse)

	for mat, count := range matCounts {
		found := 0
		newWarehouse := make([]models.Item, 0)
		for _, item := range warehouseCopy {
			if found < count && item.TypeID == mat {
				found++
			} else {
				newWarehouse = append(newWarehouse, item)
			}
		}
		if found < count {
			return false
		}
		warehouseCopy = newWarehouse
	}

	player.Warehouse = warehouseCopy

	room.SynthesisTasks = append(room.SynthesisTasks, models.SynthesisTask{
		RecipeID:     recipeID,
		StartWeek:    room.CurrentWeek,
		CompleteWeek: room.CurrentWeek + 1,
		PlayerID:     playerID,
	})

	return true
}

func (rm *RoomManager) UpgradeShop(roomID, playerID, upgradeType string) bool {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	room, ok := rm.rooms[roomID]
	if !ok || room.Phase != models.PhasePurchase {
		return false
	}

	player := room.Players[playerID]
	if player == nil || player.IsBankrupt {
		return false
	}

	switch upgradeType {
	case "shelf":
		if player.MaxShelves >= 12 {
			return false
		}
		if player.Gold < 200 {
			return false
		}
		player.Gold -= 200
		player.WeeklyStats.Expense += 200
		player.UpgradeInvestment += 200
		player.MaxShelves++
		player.Shelves = append(player.Shelves, models.ShelfSlot{
			ID:    models.NewID(),
			Price: 0,
		})
	case "decorate":
		if player.Gold < 500 {
			return false
		}
		player.Gold -= 500
		player.WeeklyStats.Expense += 500
		player.UpgradeInvestment += 500
		player.AttractionBonus += 0.2
	case "warehouse":
		if player.WarehouseCapacity >= 40 {
			return false
		}
		if player.Gold < 300 {
			return false
		}
		player.Gold -= 300
		player.WeeklyStats.Expense += 300
		player.UpgradeInvestment += 300
		player.WarehouseCapacity += 20
	case "branch":
		if len(player.BranchShops) >= 2 {
			return false
		}
		if player.Gold < 1000 {
			return false
		}
		player.Gold -= 1000
		player.WeeklyStats.Expense += 1000
		player.UpgradeInvestment += 1000
		branchShelves := make([]models.ShelfSlot, 4)
		for i := range branchShelves {
			branchShelves[i] = models.ShelfSlot{
				ID:    models.NewID(),
				Price: 0,
			}
		}
		player.BranchShops = append(player.BranchShops, models.BranchShop{
			ID:      models.NewID(),
			Shelves: branchShelves,
		})
	default:
		return false
	}

	return true
}

func (rm *RoomManager) RemoveItemFromShelf(roomID, playerID, shelfID string) bool {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	room, ok := rm.rooms[roomID]
	if !ok || room.Phase != models.PhasePurchase {
		return false
	}

	player := room.Players[playerID]
	if player == nil || player.IsBankrupt {
		return false
	}

	for i := range player.Shelves {
		if player.Shelves[i].ID == shelfID && player.Shelves[i].Item != nil {
			if len(player.Warehouse) >= player.WarehouseCapacity {
				return false
			}
			player.Warehouse = append(player.Warehouse, *player.Shelves[i].Item)
			player.Shelves[i].Item = nil
			player.Shelves[i].ItemID = ""
			player.Shelves[i].Price = 0
			return true
		}
	}

	return false
}

const (
	AuctionListingFeeRate   = 0.05
	AuctionCommissionRate   = 0.08
	AuctionMinBidIncrement  = 0.10
	AuctionDurationWeeks    = 2
	MaxActiveAuctionsPerPlayer = 5
)

func (rm *RoomManager) CreateAuction(roomID, playerID, itemID string, startingPrice, buyoutPrice int) (*models.Auction, string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	room, ok := rm.rooms[roomID]
	if !ok || room.Phase != models.PhasePurchase {
		return nil, "只能在进货日挂单"
	}

	player := room.Players[playerID]
	if player == nil || player.IsBankrupt {
		return nil, "玩家无效或已破产"
	}

	activeCount := 0
	for _, a := range room.Auctions {
		if a.SellerID == playerID && a.Status == models.AuctionActive {
			activeCount++
		}
	}
	if activeCount >= MaxActiveAuctionsPerPlayer {
		return nil, "同时最多挂5件商品"
	}

	if startingPrice <= 0 {
		return nil, "起拍价必须大于0"
	}

	if buyoutPrice > 0 && buyoutPrice <= startingPrice {
		return nil, "一口价必须大于起拍价"
	}

	listingFee := int(float64(startingPrice) * AuctionListingFeeRate)
	if listingFee < 1 {
		listingFee = 1
	}
	if player.Gold < listingFee {
		return nil, "金币不足支付挂单费"
	}

	itemIdx := -1
	for i, item := range player.Warehouse {
		if item.ID == itemID {
			itemIdx = i
			break
		}
	}
	if itemIdx == -1 {
		return nil, "仓库中未找到该商品"
	}

	item := player.Warehouse[itemIdx]
	player.Warehouse = append(player.Warehouse[:itemIdx], player.Warehouse[itemIdx+1:]...)

	player.Gold -= listingFee

	itemType, _ := models.GetItemType(item.TypeID)
	itemTypeName := ""
	if itemType.ID != "" {
		itemTypeName = itemType.Name
	}

	auction := models.Auction{
		ID:              models.NewID(),
		SellerID:        playerID,
		SellerShopName:  player.ShopName,
		Item:            item,
		ItemTypeName:    itemTypeName,
		StartingPrice:   startingPrice,
		BuyoutPrice:     buyoutPrice,
		CurrentPrice:    startingPrice,
		HighestBidderID: "",
		HighestBidderName: "",
		BidHistory:      make([]models.AuctionBid, 0),
		StartWeek:       room.CurrentWeek,
		EndWeek:         room.CurrentWeek + AuctionDurationWeeks,
		Status:          models.AuctionActive,
		CreatedAt:       time.Now().Unix(),
	}

	room.Auctions = append(room.Auctions, auction)
	return &auction, ""
}

func (rm *RoomManager) PlaceBid(roomID, playerID, auctionID string, bidAmount int) (*models.Auction, string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	room, ok := rm.rooms[roomID]
	if !ok {
		return nil, "房间不存在"
	}

	if room.Phase != models.PhaseBusiness && room.Phase != models.PhaseExplore {
		return nil, "只能在营业日或探索日出价"
	}

	player := room.Players[playerID]
	if player == nil || player.IsBankrupt {
		return nil, "玩家无效或已破产"
	}

	auctionIdx := -1
	for i, a := range room.Auctions {
		if a.ID == auctionID {
			auctionIdx = i
			break
		}
	}
	if auctionIdx == -1 {
		return nil, "拍卖不存在"
	}

	auction := &room.Auctions[auctionIdx]

	if auction.Status != models.AuctionActive {
		return nil, "拍卖已结束"
	}

	if auction.SellerID == playerID {
		return nil, "不能对自己挂出的商品出价"
	}

	minBid := auction.CurrentPrice + int(float64(auction.CurrentPrice)*AuctionMinBidIncrement)
	if auction.CurrentPrice == auction.StartingPrice && len(auction.BidHistory) == 0 {
		minBid = auction.CurrentPrice
	}

	if bidAmount < minBid {
		return nil, "出价必须高于当前最高价的10%"
	}

	if auction.BuyoutPrice > 0 && bidAmount >= auction.BuyoutPrice {
		return rm.executeBuyoutLocked(room, playerID, auctionIdx)
	}

	if player.Gold < bidAmount {
		return nil, "可用金币不足"
	}

	player.Gold -= bidAmount
	player.FrozenGold += bidAmount

	if auction.HighestBidderID != "" {
		prevBidder := room.Players[auction.HighestBidderID]
		if prevBidder != nil {
			refundAmount := auction.CurrentPrice
			prevBidder.FrozenGold -= refundAmount
			if prevBidder.FrozenGold < 0 {
				prevBidder.FrozenGold = 0
			}
			prevBidder.Gold += refundAmount
		}
	}

	auction.CurrentPrice = bidAmount
	auction.HighestBidderID = playerID
	auction.HighestBidderName = player.Name
	auction.BidHistory = append(auction.BidHistory, models.AuctionBid{
		BidderID:   playerID,
		BidderName: player.Name,
		Amount:     bidAmount,
		Timestamp:  time.Now().Unix(),
	})

	return auction, ""
}

func (rm *RoomManager) Buyout(roomID, playerID, auctionID string) (*models.Auction, string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	room, ok := rm.rooms[roomID]
	if !ok {
		return nil, "房间不存在"
	}

	if room.Phase != models.PhaseBusiness && room.Phase != models.PhaseExplore {
		return nil, "只能在营业日或探索日购买"
	}

	player := room.Players[playerID]
	if player == nil || player.IsBankrupt {
		return nil, "玩家无效或已破产"
	}

	auctionIdx := -1
	for i, a := range room.Auctions {
		if a.ID == auctionID {
			auctionIdx = i
			break
		}
	}
	if auctionIdx == -1 {
		return nil, "拍卖不存在"
	}

	auction := &room.Auctions[auctionIdx]

	if auction.Status != models.AuctionActive {
		return nil, "拍卖已结束"
	}

	if auction.SellerID == playerID {
		return nil, "不能购买自己挂出的商品"
	}

	if auction.BuyoutPrice <= 0 {
		return nil, "该拍卖无一口价"
	}

	return rm.executeBuyoutLocked(room, playerID, auctionIdx)
}

func (rm *RoomManager) executeBuyoutLocked(room *models.Room, playerID string, auctionIdx int) (*models.Auction, string) {
	auction := &room.Auctions[auctionIdx]
	player := room.Players[playerID]

	if player.Gold < auction.BuyoutPrice {
		return nil, "可用金币不足"
	}

	player.Gold -= auction.BuyoutPrice

	if auction.HighestBidderID != "" {
		prevBidder := room.Players[auction.HighestBidderID]
		if prevBidder != nil {
			refundAmount := auction.CurrentPrice
			prevBidder.FrozenGold -= refundAmount
			if prevBidder.FrozenGold < 0 {
				prevBidder.FrozenGold = 0
			}
			prevBidder.Gold += refundAmount
		}
	}

	seller := room.Players[auction.SellerID]
	commission := int(float64(auction.BuyoutPrice) * AuctionCommissionRate)
	sellerEarnings := auction.BuyoutPrice - commission

	if seller != nil {
		seller.Gold += sellerEarnings
		seller.WeeklyStats.Income += sellerEarnings
	}

	if player != nil && len(player.Warehouse) < player.WarehouseCapacity {
		player.Warehouse = append(player.Warehouse, auction.Item)
	}

	auction.CurrentPrice = auction.BuyoutPrice
	auction.HighestBidderID = playerID
	auction.HighestBidderName = player.Name
	auction.Status = models.AuctionSold
	auction.BidHistory = append(auction.BidHistory, models.AuctionBid{
		BidderID:   playerID,
		BidderName: player.Name,
		Amount:     auction.BuyoutPrice,
		Timestamp:  time.Now().Unix(),
	})

	return auction, ""
}

func (rm *RoomManager) SettleAuctions(roomID string) []models.Auction {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	room, ok := rm.rooms[roomID]
	if !ok {
		return nil
	}

	settled := make([]models.Auction, 0)

	for i := range room.Auctions {
		auction := &room.Auctions[i]
		if auction.Status != models.AuctionActive {
			continue
		}

		if room.CurrentWeek < auction.EndWeek {
			continue
		}

		if auction.HighestBidderID != "" {
			winnerID := auction.HighestBidderID
			winner := room.Players[winnerID]

			if winner != nil && winner.IsBankrupt {
				rm.findNextValidBidder(room, auction)
				winnerID = auction.HighestBidderID
				winner = room.Players[winnerID]
			}

			if winnerID != "" && winner != nil && !winner.IsBankrupt {
				winner.FrozenGold -= auction.CurrentPrice
				if winner.FrozenGold < 0 {
					winner.FrozenGold = 0
				}

				if len(winner.Warehouse) < winner.WarehouseCapacity {
					winner.Warehouse = append(winner.Warehouse, auction.Item)
				}

				seller := room.Players[auction.SellerID]
				commission := int(float64(auction.CurrentPrice) * AuctionCommissionRate)
				sellerEarnings := auction.CurrentPrice - commission

				if seller != nil {
					seller.Gold += sellerEarnings
					seller.WeeklyStats.Income += sellerEarnings
				}

				auction.Status = models.AuctionSold
				settled = append(settled, *auction)
			} else {
				rm.refundAllBidders(room, auction)

				seller := room.Players[auction.SellerID]
				if seller != nil && len(seller.Warehouse) < seller.WarehouseCapacity {
					seller.Warehouse = append(seller.Warehouse, auction.Item)
				}

				auction.Status = models.AuctionExpired
				settled = append(settled, *auction)
			}
		} else {
			seller := room.Players[auction.SellerID]
			if seller != nil && len(seller.Warehouse) < seller.WarehouseCapacity {
				seller.Warehouse = append(seller.Warehouse, auction.Item)
			}

			auction.Status = models.AuctionExpired
			settled = append(settled, *auction)
		}
	}

	return settled
}

func (rm *RoomManager) findNextValidBidder(room *models.Room, auction *models.Auction) {
	for i := len(auction.BidHistory) - 1; i >= 0; i-- {
		bid := auction.BidHistory[i]
		bidder := room.Players[bid.BidderID]
		if bidder != nil && !bidder.IsBankrupt {
			if auction.HighestBidderID != "" && auction.HighestBidderID != bid.BidderID {
				prevBidder := room.Players[auction.HighestBidderID]
				if prevBidder != nil {
					refundAmount := auction.CurrentPrice
					prevBidder.FrozenGold -= refundAmount
					if prevBidder.FrozenGold < 0 {
						prevBidder.FrozenGold = 0
					}
					prevBidder.Gold += refundAmount
				}
			}

			auction.HighestBidderID = bid.BidderID
			auction.HighestBidderName = bid.BidderName
			auction.CurrentPrice = bid.Amount

			player := room.Players[bid.BidderID]
			if player != nil {
				player.FrozenGold += bid.Amount
				player.Gold -= bid.Amount
			}
			return
		}
	}

	if auction.HighestBidderID != "" {
		prevBidder := room.Players[auction.HighestBidderID]
		if prevBidder != nil {
			refundAmount := auction.CurrentPrice
			prevBidder.FrozenGold -= refundAmount
			if prevBidder.FrozenGold < 0 {
				prevBidder.FrozenGold = 0
			}
			prevBidder.Gold += refundAmount
		}
	}

	auction.HighestBidderID = ""
	auction.HighestBidderName = ""
	auction.CurrentPrice = auction.StartingPrice
}

func (rm *RoomManager) refundAllBidders(room *models.Room, auction *models.Auction) {
	if auction.HighestBidderID != "" {
		bidder := room.Players[auction.HighestBidderID]
		if bidder != nil {
			bidder.FrozenGold -= auction.CurrentPrice
			if bidder.FrozenGold < 0 {
				extraRefund := -bidder.FrozenGold
				bidder.FrozenGold = 0
				bidder.Gold += extraRefund
			}
		}
	}
}

func (rm *RoomManager) CancelAuction(roomID, playerID, auctionID string) (*models.Auction, string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	room, ok := rm.rooms[roomID]
	if !ok {
		return nil, "房间不存在"
	}

	auctionIdx := -1
	for i, a := range room.Auctions {
		if a.ID == auctionID {
			auctionIdx = i
			break
		}
	}
	if auctionIdx == -1 {
		return nil, "拍卖不存在"
	}

	auction := &room.Auctions[auctionIdx]

	if auction.SellerID != playerID {
		return nil, "只能取消自己的拍卖"
	}

	if auction.Status != models.AuctionActive {
		return nil, "拍卖已结束，无法取消"
	}

	if len(auction.BidHistory) > 0 {
		return nil, "已有人出价，无法取消"
	}

	player := room.Players[playerID]
	if player != nil && len(player.Warehouse) < player.WarehouseCapacity {
		player.Warehouse = append(player.Warehouse, auction.Item)
	}

	auction.Status = models.AuctionCancelled
	return auction, ""
}

func (rm *RoomManager) settleAuctionsLocked(room *models.Room) {
	for i := range room.Auctions {
		auction := &room.Auctions[i]
		if auction.Status != models.AuctionActive {
			continue
		}

		if room.CurrentWeek < auction.EndWeek {
			continue
		}

		if auction.HighestBidderID != "" {
			winnerID := auction.HighestBidderID
			winner := room.Players[winnerID]

			if winner != nil && winner.IsBankrupt {
				rm.findNextValidBidder(room, auction)
				winnerID = auction.HighestBidderID
				winner = room.Players[winnerID]
			}

			if winnerID != "" && winner != nil && !winner.IsBankrupt {
				winner.FrozenGold -= auction.CurrentPrice
				if winner.FrozenGold < 0 {
					winner.FrozenGold = 0
				}

				if len(winner.Warehouse) < winner.WarehouseCapacity {
					winner.Warehouse = append(winner.Warehouse, auction.Item)
				}

				seller := room.Players[auction.SellerID]
				commission := int(float64(auction.CurrentPrice) * AuctionCommissionRate)
				sellerEarnings := auction.CurrentPrice - commission

				if seller != nil {
					seller.Gold += sellerEarnings
					seller.WeeklyStats.Income += sellerEarnings
				}

				auction.Status = models.AuctionSold
			} else {
				rm.refundAllBidders(room, auction)

				seller := room.Players[auction.SellerID]
				if seller != nil && len(seller.Warehouse) < seller.WarehouseCapacity {
					seller.Warehouse = append(seller.Warehouse, auction.Item)
				}

				auction.Status = models.AuctionExpired
			}
		} else {
			seller := room.Players[auction.SellerID]
			if seller != nil && len(seller.Warehouse) < seller.WarehouseCapacity {
				seller.Warehouse = append(seller.Warehouse, auction.Item)
			}

			auction.Status = models.AuctionExpired
		}
	}
}
