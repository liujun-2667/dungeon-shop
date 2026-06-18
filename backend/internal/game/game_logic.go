package game

import (
	"math/rand"

	"dungeon-shop/internal/models"
)

type SeededRand struct {
	r *rand.Rand
}

func NewSeededRand(seed int64) *SeededRand {
	return &SeededRand{r: rand.New(rand.NewSource(seed))}
}

func (sr *SeededRand) Intn(n int) int {
	return sr.r.Intn(n)
}

func (sr *SeededRand) Float64() float64 {
	return sr.r.Float64()
}

func (sr *SeededRand) Perm(n int) []int {
	return sr.r.Perm(n)
}

func GenerateWholesalerStock(seed int64, week int) []models.WholesalerItem {
	sr := NewSeededRand(seed + int64(week)*1000)
	itemTypeIDs := make([]string, 0, len(models.ItemTypes))
	for id := range models.ItemTypes {
		itemTypeIDs = append(itemTypeIDs, id)
	}

	qualities := []models.Quality{
		models.QualityCommon,
		models.QualityCommon,
		models.QualityCommon,
		models.QualityFine,
		models.QualityFine,
		models.QualityRare,
	}

	if week >= 6 {
		qualities = append(qualities, models.QualityRare)
	}
	if week >= 10 {
		qualities = append(qualities, models.QualityLegendary)
	}

	stock := make([]models.WholesalerItem, 0)
	numItems := 8 + sr.Intn(6)

	for i := 0; i < numItems; i++ {
		typeID := itemTypeIDs[sr.Intn(len(itemTypeIDs))]
		itemType := models.ItemTypes[typeID]
		quality := qualities[sr.Intn(len(qualities))]

		baseCost := models.CalculatePrice(itemType.BasePrice, quality)
		price := int(float64(baseCost) * (0.8 + sr.Float64()*0.4))
		quantity := 1 + sr.Intn(4)

		stock = append(stock, models.WholesalerItem{
			TypeID:   typeID,
			Quality:  quality,
			Price:    price,
			Quantity: quantity,
		})
	}

	return stock
}

func GenerateNPCs(seed int64, week int, playerCount int, playerIDs []string, event *models.GlobalEvent) []models.NPC {
	sr := NewSeededRand(seed + int64(week)*2000 + 500)
	numNPCs := 8 + week*2

	classes := []models.NPCClass{models.ClassWarrior, models.ClassMage, models.ClassRogue}
	npcs := make([]models.NPC, 0, numNPCs)

	for i := 0; i < numNPCs; i++ {
		class := classes[sr.Intn(len(classes))]
		names := models.NPCNames[class]
		name := names[sr.Intn(len(names))]

		baseBudget := 30 + week*5 + sr.Intn(20)
		budget := baseBudget

		preferences := map[models.Category]float64{
			models.CategoryWeapon:   0.5,
			models.CategoryArmor:    0.5,
			models.CategoryConsumable: 0.5,
			models.CategoryMaterial: 0.3,
		}

		switch class {
		case models.ClassWarrior:
			preferences[models.CategoryWeapon] = 1.0
			preferences[models.CategoryArmor] = 0.9
			budget = int(float64(baseBudget) * 1.2)
		case models.ClassMage:
			preferences[models.CategoryConsumable] = 1.0
			preferences[models.CategoryWeapon] = 0.7
			budget = int(float64(baseBudget) * 1.1)
		case models.ClassRogue:
			preferences[models.CategoryConsumable] = 0.9
			preferences[models.CategoryArmor] = 0.6
			budget = int(float64(baseBudget) * 0.9)
		}

		if event != nil {
			for cat, mult := range event.Effects.DemandMultiplier {
				preferences[cat] *= mult
			}
		}

		npc := models.NPC{
			ID:                models.NewID(),
			Name:              name,
			Class:             class,
			Budget:            budget,
			Preferences:       preferences,
			IsVIP:             false,
			PriceSensitivity:  0.6 + sr.Float64()*0.6,
			MaxShopsToVisit:   1 + sr.Intn(playerCount),
			Impulsiveness:     0.3 + sr.Float64()*0.7,
			QualityPreference: 0.3 + sr.Float64()*0.7,
		}

		npcs = append(npcs, npc)
	}

	if event != nil && event.Effects.VIPCustomer && len(playerIDs) > 0 {
		nonBankruptIDs := make([]string, 0)
		if len(playerIDs) > 0 {
			nonBankruptIDs = playerIDs
		}
		if len(nonBankruptIDs) == 0 {
			nonBankruptIDs = playerIDs
		}
		targetIdx := sr.Intn(len(nonBankruptIDs))
		targetID := nonBankruptIDs[targetIdx]

		vipBudget := 500 + week*50
		vip := models.NPC{
			ID:     models.NewID(),
			Name:   "贵族大人",
			Class:  models.ClassWarrior,
			Budget: vipBudget,
			Preferences: map[models.Category]float64{
				models.CategoryWeapon:     1.5,
				models.CategoryArmor:      1.5,
				models.CategoryConsumable: 1.0,
				models.CategoryMaterial:   1.0,
			},
			IsVIP:             true,
			TargetPlayerID:    targetID,
			PriceSensitivity:  0.2,
			MaxShopsToVisit:   1,
			Impulsiveness:     0.8,
			QualityPreference: 1.0,
		}
		npcs = append(npcs, vip)
	}

	return npcs
}

func GenerateRandomEvent(seed int64, week int) *models.GlobalEvent {
	sr := NewSeededRand(seed + int64(week)*3000 + 1000)

	if sr.Float64() > 0.3 {
		return nil
	}

	eventType := models.EventTypes[sr.Intn(len(models.EventTypes))]
	info := models.EventInfo[eventType]
	duration := 1 + sr.Intn(2)

	event := &models.GlobalEvent{
		ID:          models.NewID(),
		Type:        eventType,
		Name:        info.Name,
		Description: info.Description,
		Duration:    duration,
		StartWeek:   week,
		Effects:     models.EventEffects{},
	}

	switch eventType {
	case "plague":
		event.Effects.DemandMultiplier = map[models.Category]float64{
			models.CategoryConsumable: 2.0,
		}
		event.Effects.PriceMultiplier = map[models.Category]float64{
			models.CategoryConsumable: 1.5,
		}
	case "war":
		event.Effects.DemandMultiplier = map[models.Category]float64{
			models.CategoryWeapon: 3.0,
			models.CategoryArmor:  2.0,
		}
	case "harvest":
		event.Effects.PriceMultiplier = map[models.Category]float64{
			models.CategoryConsumable: 0.5,
		}
	case "thieves":
		event.Effects.StealItem = true
	case "noble":
		event.Effects.VIPCustomer = true
	case "cavein":
		event.Effects.BlockExploration = true
	}

	return event
}

func GenerateRecipes(seed int64, count int) []models.Recipe {
	sr := NewSeededRand(seed)
	materials := []string{"ore", "leather", "crystal", "herb"}
	outputs := []string{"sword", "bow", "staff", "axe", "shield", "armor", "boots", "helmet", "potion", "scroll"}
	qualities := []models.Quality{models.QualityFine, models.QualityRare}

	recipes := make([]models.Recipe, 0, count)

	for i := 0; i < count; i++ {
		numMats := 2 + sr.Intn(3)
		matSet := make(map[string]bool)
		recipeMats := make([]string, 0, numMats)

		for len(recipeMats) < numMats {
			mat := materials[sr.Intn(len(materials))]
			if !matSet[mat] {
				matSet[mat] = true
				recipeMats = append(recipeMats, mat)
			}
		}

		output := outputs[sr.Intn(len(outputs))]
		quality := qualities[sr.Intn(len(qualities))]

		recipes = append(recipes, models.Recipe{
			ID:             models.NewID(),
			Name:           "神秘配方",
			Materials:      recipeMats,
			OutputItemType: output,
			OutputQuality:  quality,
		})
	}

	return recipes
}

func GenerateAdventurer(seed int64, idx int) models.Adventurer {
	sr := NewSeededRand(seed + int64(idx)*100)
	level := 1 + sr.Intn(5)
	name := models.AdventurerNames[sr.Intn(len(models.AdventurerNames))]

	return models.Adventurer{
		ID:               models.NewID(),
		Name:             name,
		Level:            level,
		HireCost:         models.GetAdventurerHireCost(level),
		IsInjured:        false,
		InjuredUntilWeek: 0,
		IsOnMission:      false,
	}
}

func CalculatePurchaseProbability(npc models.NPC, itemType models.ItemType, price int, event *models.GlobalEvent) float64 {
	basePrice := models.CalculatePrice(itemType.BasePrice, models.QualityCommon)

	priceRatio := float64(price) / float64(basePrice)

	priceMult := 1.0
	if event != nil && event.Effects.PriceMultiplier != nil {
		if m, ok := event.Effects.PriceMultiplier[itemType.Category]; ok {
			priceMult = m
		}
	}

	if priceRatio < 0.8 {
		return 1.0
	}
	if priceRatio > 1.5 {
		return 0.1
	}

	pref := npc.Preferences[itemType.Category]
	prob := pref * (1.5 - priceRatio) * priceMult

	if prob < 0 {
		prob = 0
	}
	if prob > 1 {
		prob = 1
	}

	return prob
}

func CalculateExplorationSuccessRate(adventurerLevel int, floor int) float64 {
	rate := float64(adventurerLevel)*0.2 - float64(floor)*0.1
	if rate < 0.1 {
		rate = 0.1
	}
	if rate > 0.9 {
		rate = 0.9
	}
	return rate
}

func GetFloorLoot(floor int, seed int64, week int) []models.Item {
	sr := NewSeededRand(seed + int64(week)*100 + int64(floor)*10)
	loot := make([]models.Item, 0)

	materials := []string{"ore", "leather", "crystal", "herb"}
	numItems := 1 + sr.Intn(floor)

	for i := 0; i < numItems; i++ {
		typeID := materials[sr.Intn(len(materials))]
		itemType := models.ItemTypes[typeID]

		quality := models.QualityCommon
		roll := sr.Float64()
		if floor >= 3 && roll < 0.3 {
			quality = models.QualityFine
		}
		if floor >= 4 && roll < 0.15 {
			quality = models.QualityRare
		}
		if floor >= 5 && roll < 0.05 {
			quality = models.QualityLegendary
		}

		loot = append(loot, models.Item{
			ID:           models.NewID(),
			TypeID:       typeID,
			Quality:      quality,
			PurchaseCost: int(float64(itemType.BasePrice) * models.QualityMultiplier[quality] * 0.5),
		})
	}

	return loot
}

func CalculateTotalAssets(player *models.PlayerState) int {
	inventoryValue := 0

	for _, item := range player.Warehouse {
		if itemType, ok := models.GetItemType(item.TypeID); ok {
			basePrice := models.CalculatePrice(itemType.BasePrice, item.Quality)
			inventoryValue += int(float64(basePrice) * 0.8)
		}
	}

	for _, slot := range player.Shelves {
		if slot.Item != nil {
			if itemType, ok := models.GetItemType(slot.Item.TypeID); ok {
				basePrice := models.CalculatePrice(itemType.BasePrice, slot.Item.Quality)
				inventoryValue += int(float64(basePrice) * 0.8)
			}
		}
	}

	upgradeValue := int(float64(player.UpgradeInvestment) * 0.5)

	return player.Gold + player.FrozenGold + inventoryValue + upgradeValue
}

func CheckBankruptcy(player *models.PlayerState) bool {
	if player.Gold <= 0 && player.FrozenGold <= 0 && len(player.Warehouse) == 0 {
		hasItems := false
		for _, slot := range player.Shelves {
			if slot.Item != nil {
				hasItems = true
				break
			}
		}
		if !hasItems {
			player.IsBankrupt = true
			return true
		}
	}
	return false
}

func ProcessExpiredItems(player *models.PlayerState, currentWeek int) int {
	expiredOnShelf := 0

	validItems := make([]models.Item, 0)
	for _, item := range player.Warehouse {
		itemType, ok := models.GetItemType(item.TypeID)
		if !ok || !itemType.HasShelfLife || item.ExpiresWeek > currentWeek {
			validItems = append(validItems, item)
		}
	}
	player.Warehouse = validItems

	for i := range player.Shelves {
		if player.Shelves[i].Item != nil {
			itemType, ok := models.GetItemType(player.Shelves[i].Item.TypeID)
			if ok && itemType.HasShelfLife && player.Shelves[i].Item.ExpiresWeek <= currentWeek {
				expiredOnShelf++
				player.Shelves[i].Item = nil
				player.Shelves[i].ItemID = ""
				player.Shelves[i].Price = 0
			}
		}
	}

	for bi := range player.BranchShops {
		for i := range player.BranchShops[bi].Shelves {
			if player.BranchShops[bi].Shelves[i].Item != nil {
				itemType, ok := models.GetItemType(player.BranchShops[bi].Shelves[i].Item.TypeID)
				if ok && itemType.HasShelfLife && player.BranchShops[bi].Shelves[i].Item.ExpiresWeek <= currentWeek {
					expiredOnShelf++
					player.BranchShops[bi].Shelves[i].Item = nil
					player.BranchShops[bi].Shelves[i].ItemID = ""
					player.BranchShops[bi].Shelves[i].Price = 0
				}
			}
		}
	}

	return expiredOnShelf
}

type PriceTier string

const (
	PriceTierBargain  PriceTier = "bargain"
	PriceTierFair     PriceTier = "fair"
	PriceTierOverpriced PriceTier = "overpriced"
)

func GetPriceTier(price int, basePrice int) PriceTier {
	if basePrice <= 0 {
		return PriceTierFair
	}
	ratio := float64(price) / float64(basePrice)
	if ratio <= 1.2 {
		return PriceTierBargain
	} else if ratio >= 1.4 {
		return PriceTierOverpriced
	}
	return PriceTierFair
}

func ShuffleNPCs(npcs []models.NPC, seed int64, week int) []models.NPC {
	sr := NewSeededRand(seed + int64(week)*5000)
	shuffled := make([]models.NPC, len(npcs))
	copy(shuffled, npcs)

	for i := len(shuffled) - 1; i > 0; i-- {
		j := sr.Intn(i + 1)
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}

	return shuffled
}

func GetAllPlayerPrices(room *models.Room, itemTypeID string) map[string]int {
	prices := make(map[string]int)
	for playerID, player := range room.Players {
		if player.IsBankrupt {
			continue
		}
		for _, slot := range player.Shelves {
			if slot.Item != nil && slot.Item.TypeID == itemTypeID && slot.Price > 0 {
				prices[playerID] = slot.Price
				break
			}
		}
	}
	return prices
}
