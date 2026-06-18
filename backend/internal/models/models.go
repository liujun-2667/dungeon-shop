package models

import (
	"time"

	"github.com/google/uuid"
)

type Quality string

const (
	QualityCommon    Quality = "common"
	QualityFine      Quality = "fine"
	QualityRare      Quality = "rare"
	QualityLegendary Quality = "legendary"
)

type Category string

const (
	CategoryWeapon   Category = "weapon"
	CategoryArmor    Category = "armor"
	CategoryConsumable Category = "consumable"
	CategoryMaterial Category = "material"
)

type ItemType struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Category Category `json:"category"`
	BasePrice int     `json:"basePrice"`
	HasShelfLife bool `json:"hasShelfLife"`
	ShelfLifeWeeks int `json:"shelfLifeWeeks,omitempty"`
}

type Item struct {
	ID          string  `json:"id"`
	TypeID      string  `json:"typeId"`
	Quality     Quality `json:"quality"`
	PurchaseCost int    `json:"purchaseCost"`
	ExpiresWeek int    `json:"expiresWeek,omitempty"`
}

type ShelfSlot struct {
	ID       string `json:"id"`
	ItemID   string `json:"itemId,omitempty"`
	Price    int    `json:"price"`
	Item     *Item  `json:"item,omitempty"`
}

type NPCClass string

const (
	ClassWarrior NPCClass = "warrior"
	ClassMage    NPCClass = "mage"
	ClassRogue   NPCClass = "rogue"
)

type NPC struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Class    NPCClass `json:"class"`
	Budget   int      `json:"budget"`
	Preferences map[Category]float64 `json:"preferences"`
	IsVIP    bool     `json:"isVIP"`
	TargetPlayerID string `json:"targetPlayerId,omitempty"`
	PriceSensitivity float64 `json:"priceSensitivity"`
	MaxShopsToVisit int   `json:"maxShopsToVisit"`
	Impulsiveness  float64 `json:"impulsiveness"`
	QualityPreference float64 `json:"qualityPreference"`
}

type Adventurer struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Level      int    `json:"level"`
	HireCost   int    `json:"hireCost"`
	IsInjured  bool   `json:"isInjured"`
	InjuredUntilWeek int `json:"injuredUntilWeek"`
	IsOnMission bool  `json:"isOnMission"`
}

type Recipe struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Materials []string `json:"materials"`
	OutputItemType string `json:"outputItemType"`
	OutputQuality Quality `json:"outputQuality"`
	OwnerID   string   `json:"ownerId"`
}

type AuctionStatus string

const (
	AuctionActive  AuctionStatus = "active"
	AuctionSold    AuctionStatus = "sold"
	AuctionExpired AuctionStatus = "expired"
	AuctionCancelled AuctionStatus = "cancelled"
)

type AuctionBid struct {
	BidderID   string `json:"bidderId"`
	BidderName string `json:"bidderName"`
	Amount     int    `json:"amount"`
	Timestamp  int64  `json:"timestamp"`
}

type Auction struct {
	ID             string        `json:"id"`
	SellerID       string        `json:"sellerId"`
	SellerShopName string        `json:"sellerShopName"`
	Item           Item          `json:"item"`
	ItemTypeName   string        `json:"itemTypeName"`
	StartingPrice  int           `json:"startingPrice"`
	BuyoutPrice    int           `json:"buyoutPrice,omitempty"`
	CurrentPrice   int           `json:"currentPrice"`
	HighestBidderID string       `json:"highestBidderId,omitempty"`
	HighestBidderName string     `json:"highestBidderName,omitempty"`
	BidHistory     []AuctionBid  `json:"bidHistory"`
	StartWeek      int           `json:"startWeek"`
	EndWeek        int           `json:"endWeek"`
	Status         AuctionStatus `json:"status"`
	CreatedAt      int64         `json:"createdAt"`
}

type PlayerState struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	ShopName   string      `json:"shopName"`
	Gold       int         `json:"gold"`
	FrozenGold int         `json:"frozenGold"`
	IsBankrupt bool        `json:"isBankrupt"`
	Shelves    []ShelfSlot `json:"shelves"`
	MaxShelves int         `json:"maxShelves"`
	Warehouse  []Item      `json:"warehouse"`
	WarehouseCapacity int  `json:"warehouseCapacity"`
	Adventurers []Adventurer `json:"adventurers"`
	MaxAdventurers int     `json:"maxAdventurers"`
	Recipes    []Recipe    `json:"recipes"`
	BranchShops []BranchShop `json:"branchShops"`
	AttractionBonus float64 `json:"attractionBonus"`
	UpgradeInvestment int `json:"upgradeInvestment"`
	WeeklyStats  WeeklyStats `json:"weeklyStats"`
	AssetHistory []int       `json:"assetHistory"`
	Reputation  int         `json:"reputation"`
}

type BranchShop struct {
	ID      string      `json:"id"`
	Shelves []ShelfSlot `json:"shelves"`
}

type WeeklyStats struct {
	Income      int `json:"income"`
	Expense     int `json:"expense"`
	ItemsSold   int `json:"itemsSold"`
	ItemsBought int `json:"itemsBought"`
}

type GamePhase string

const (
	PhasePurchase  GamePhase = "purchase"
	PhaseBusiness  GamePhase = "business"
	PhaseExplore   GamePhase = "explore"
	PhaseSettlement GamePhase = "settlement"
)

type GlobalEvent struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Name       string `json:"name"`
	Description string `json:"description"`
	Duration   int    `json:"duration"`
	StartWeek  int    `json:"startWeek"`
	Effects    EventEffects `json:"effects"`
}

type EventEffects struct {
	DemandMultiplier map[Category]float64 `json:"demandMultiplier,omitempty"`
	PriceMultiplier  map[Category]float64 `json:"priceMultiplier,omitempty"`
	StealItem        bool                 `json:"stealItem,omitempty"`
	VIPCustomer      bool                 `json:"vipCustomer,omitempty"`
	BlockExploration bool                 `json:"blockExploration,omitempty"`
}

type WholesalerItem struct {
	TypeID    string  `json:"typeId"`
	Quality   Quality `json:"quality"`
	Price     int     `json:"price"`
	Quantity  int     `json:"quantity"`
}

type ExplorationMission struct {
	PlayerID     string `json:"playerId"`
	AdventurerID string `json:"adventurerId"`
	Floor        int    `json:"floor"`
	Week         int    `json:"week"`
}

type SynthesisTask struct {
	RecipeID   string `json:"recipeId"`
	StartWeek  int    `json:"startWeek"`
	CompleteWeek int  `json:"completeWeek"`
	PlayerID   string `json:"playerId"`
}

type Room struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Players      map[string]*PlayerState `json:"players"`
	MaxPlayers   int                    `json:"maxPlayers"`
	CurrentWeek  int                    `json:"currentWeek"`
	TotalWeeks   int                    `json:"totalWeeks"`
	Phase        GamePhase              `json:"phase"`
	PhaseEndTime int64                  `json:"phaseEndTime"`
	PhaseDuration int                   `json:"phaseDuration"`
	Seed         int64                  `json:"seed"`
	WholesalerStock []WholesalerItem    `json:"wholesalerStock"`
	NPCsThisWeek []NPC                  `json:"npcsThisWeek"`
	CurrentEvent *GlobalEvent           `json:"currentEvent"`
	SynthesisTasks []SynthesisTask       `json:"synthesisTasks"`
	ExplorationMissions []ExplorationMission `json:"explorationMissions"`
	Auctions      []Auction             `json:"auctions"`
	Status       string                 `json:"status"`
	CreatedAt    time.Time              `json:"createdAt"`

	NPCIndex       int                    `json:"-"`
	PendingBargain *BargainRequest        `json:"-"`
	BargainNPCIdx  int                    `json:"-"`
	BargainSlot    *ShelfSlot             `json:"-"`
	BargainNPC     *NPC                   `json:"-"`
	BargainPlayerID string                 `json:"-"`
	BusinessLogs   []BusinessLogEntry     `json:"-"`
}

type BusinessLogEntry struct {
	PlayerID string `json:"playerId"`
	NPCName  string `json:"npcName"`
	Message  string `json:"message"`
	Type     string `json:"type"`
}

type PlayerProfile struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	TotalGames   int       `json:"totalGames"`
	Wins         int       `json:"wins"`
	TotalEarnings int64    `json:"totalEarnings"`
	CreatedAt    time.Time `json:"createdAt"`
}

type GameRecord struct {
	ID         string    `json:"id"`
	RoomID     string    `json:"roomId"`
	StartTime  time.Time `json:"startTime"`
	EndTime    time.Time `json:"endTime"`
	Players    []GamePlayerResult `json:"players"`
	Seed       int64     `json:"seed"`
}

type GamePlayerResult struct {
	PlayerID string `json:"playerId"`
	Name     string `json:"name"`
	FinalAssets int `json:"finalAssets"`
	Rank     int `json:"rank"`
	IsWinner bool `json:"isWinner"`
}

type WSMessage struct {
	Type    string      `json:"type"`
	Data    interface{} `json:"data"`
	RoomID  string      `json:"roomId,omitempty"`
	PlayerID string     `json:"playerId,omitempty"`
}

type BargainRequest struct {
	ID          string `json:"id"`
	NPCID       string `json:"npcId"`
	NPCName     string `json:"npcName"`
	NPCClass    NPCClass `json:"npcClass"`
	ShelfID     string `json:"shelfId"`
	ItemTypeID  string `json:"itemTypeId"`
	ItemName    string `json:"itemName"`
	ItemQuality Quality `json:"itemQuality"`
	OriginalPrice int `json:"originalPrice"`
	BargainedPrice int `json:"bargainedPrice"`
	ExpiresAt   int64  `json:"expiresAt"`
	Resolved    bool   `json:"-"`
}

type ReputationLevel string

const (
	ReputationHonest  ReputationLevel = "honest"
	ReputationNormal  ReputationLevel = "normal"
	ReputationShady   ReputationLevel = "shady"
)

const (
	ReputationHonestThreshold = 5
	ReputationShadyThreshold  = -5
)

func GetReputationLevel(reputation int) ReputationLevel {
	if reputation >= ReputationHonestThreshold {
		return ReputationHonest
	} else if reputation <= ReputationShadyThreshold {
		return ReputationShady
	}
	return ReputationNormal
}

func GetReputationLabel(level ReputationLevel) string {
	switch level {
	case ReputationHonest:
		return "信誉商铺"
	case ReputationShady:
		return "黑心店铺"
	default:
		return "普通店铺"
	}
}

func GetReputationVisitBonus(level ReputationLevel) float64 {
	switch level {
	case ReputationHonest:
		return 0.3
	case ReputationShady:
		return -0.4
	default:
		return 0.0
	}
}

func GetReputationBudgetBonus(level ReputationLevel) float64 {
	switch level {
	case ReputationHonest:
		return 0.1
	case ReputationShady:
		return -0.2
	default:
		return 0.0
	}
}

func NewID() string {
	return uuid.New().String()
}

func NewPlayerState(name, shopName string) *PlayerState {
	shelves := make([]ShelfSlot, 6)
	for i := range shelves {
		shelves[i] = ShelfSlot{
			ID:    NewID(),
			Price: 0,
		}
	}

	return &PlayerState{
		ID:                NewID(),
		Name:              name,
		ShopName:          shopName,
		Gold:              500,
		FrozenGold:        0,
		IsBankrupt:        false,
		Shelves:           shelves,
		MaxShelves:        6,
		Warehouse:         make([]Item, 0),
		WarehouseCapacity: 20,
		Adventurers:       make([]Adventurer, 0),
		MaxAdventurers:    3,
		Recipes:           make([]Recipe, 0),
		BranchShops:       make([]BranchShop, 0),
		AttractionBonus:   1.0,
		UpgradeInvestment: 0,
		WeeklyStats:       WeeklyStats{},
		AssetHistory:      []int{500},
		Reputation:        0,
	}
}

func NewRoom(name string, maxPlayers int, seed int64) *Room {
	return &Room{
		ID:                  NewID(),
		Name:                name,
		Players:             make(map[string]*PlayerState),
		MaxPlayers:          maxPlayers,
		CurrentWeek:         0,
		TotalWeeks:          12,
		Phase:               PhasePurchase,
		PhaseDuration:       15,
		Seed:                seed,
		WholesalerStock:     make([]WholesalerItem, 0),
		NPCsThisWeek:        make([]NPC, 0),
		SynthesisTasks:      make([]SynthesisTask, 0),
		ExplorationMissions: make([]ExplorationMission, 0),
		Auctions:            make([]Auction, 0),
		Status:              "waiting",
		CreatedAt:           time.Now(),
	}
}
