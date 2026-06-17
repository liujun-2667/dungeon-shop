package models

var ItemTypes = map[string]ItemType{
	"sword": {
		ID:         "sword",
		Name:       "长剑",
		Category:   CategoryWeapon,
		BasePrice:  50,
		HasShelfLife: false,
	},
	"bow": {
		ID:         "bow",
		Name:       "长弓",
		Category:   CategoryWeapon,
		BasePrice:  45,
		HasShelfLife: false,
	},
	"staff": {
		ID:         "staff",
		Name:       "法杖",
		Category:   CategoryWeapon,
		BasePrice:  60,
		HasShelfLife: false,
	},
	"axe": {
		ID:         "axe",
		Name:       "战斧",
		Category:   CategoryWeapon,
		BasePrice:  55,
		HasShelfLife: false,
	},
	"shield": {
		ID:         "shield",
		Name:       "盾牌",
		Category:   CategoryArmor,
		BasePrice:  40,
		HasShelfLife: false,
	},
	"armor": {
		ID:         "armor",
		Name:       "铠甲",
		Category:   CategoryArmor,
		BasePrice:  80,
		HasShelfLife: false,
	},
	"boots": {
		ID:         "boots",
		Name:       "战靴",
		Category:   CategoryArmor,
		BasePrice:  35,
		HasShelfLife: false,
	},
	"helmet": {
		ID:         "helmet",
		Name:       "头盔",
		Category:   CategoryArmor,
		BasePrice:  45,
		HasShelfLife: false,
	},
	"potion": {
		ID:         "potion",
		Name:       "治疗药水",
		Category:   CategoryConsumable,
		BasePrice:  20,
		HasShelfLife: true,
		ShelfLifeWeeks: 3,
	},
	"scroll": {
		ID:         "scroll",
		Name:       "魔法卷轴",
		Category:   CategoryConsumable,
		BasePrice:  30,
		HasShelfLife: true,
		ShelfLifeWeeks: 3,
	},
	"trap": {
		ID:         "trap",
		Name:       "陷阱",
		Category:   CategoryConsumable,
		BasePrice:  25,
		HasShelfLife: true,
		ShelfLifeWeeks: 3,
	},
	"food": {
		ID:         "food",
		Name:       "口粮",
		Category:   CategoryConsumable,
		BasePrice:  10,
		HasShelfLife: true,
		ShelfLifeWeeks: 3,
	},
	"ore": {
		ID:         "ore",
		Name:       "精铁矿石",
		Category:   CategoryMaterial,
		BasePrice:  15,
		HasShelfLife: false,
	},
	"leather": {
		ID:         "leather",
		Name:       "魔兽皮革",
		Category:   CategoryMaterial,
		BasePrice:  12,
		HasShelfLife: false,
	},
	"crystal": {
		ID:         "crystal",
		Name:       "魔晶",
		Category:   CategoryMaterial,
		BasePrice:  25,
		HasShelfLife: false,
	},
	"herb": {
		ID:         "herb",
		Name:       "魔法草药",
		Category:   CategoryMaterial,
		BasePrice:  18,
		HasShelfLife: false,
	},
}

var QualityMultiplier = map[Quality]float64{
	QualityCommon:    1.0,
	QualityFine:      1.5,
	QualityRare:      2.5,
	QualityLegendary: 4.0,
}

var NPCNames = map[NPCClass][]string{
	ClassWarrior: {"铁壁", "狂战士", "守护骑士", "武器大师", "角斗士", "圣骑士"},
	ClassMage:    {"火焰法师", "冰霜术士", "奥术师", "元素使", "召唤师", "咒术师"},
	ClassRogue:   {"暗影刺客", "盗贼", "游侠", "赏金猎人", "忍者", "破坏者"},
}

var AdventurerNames = []string{"艾伦", "贝拉", "卡洛斯", "黛安娜", "伊森", "菲奥娜", "加文", "海蒂", "艾萨克", "茱莉亚"}

var EventTypes = []string{"plague", "war", "harvest", "thieves", "noble", "cavein"}

var EventInfo = map[string]struct {
	Name        string
	Description string
}{
	"plague": {
		Name:        "瘟疫",
		Description: "药水需求翻倍，价格可上涨",
	},
	"war": {
		Name:        "战争",
		Description: "武器需求激增",
	},
	"harvest": {
		Name:        "丰收",
		Description: "食粮过剩，售价暴跌",
	},
	"thieves": {
		Name:        "盗贼横行",
		Description: "每家店丢失一件随机商品",
	},
	"noble": {
		Name:        "贵族来访",
		Description: "出现一位预算极高的VIP顾客",
	},
	"cavein": {
		Name:        "地牢塌方",
		Description: "本周探索全部失败",
	},
}

func GetItemType(typeID string) (ItemType, bool) {
	it, ok := ItemTypes[typeID]
	return it, ok
}

func CalculatePrice(basePrice int, quality Quality) int {
	mult := QualityMultiplier[quality]
	return int(float64(basePrice) * mult)
}

func GetAdventurerHireCost(level int) int {
	return 50 + (level-1)*30
}

func (q Quality) Rank() int {
	switch q {
	case QualityCommon:
		return 0
	case QualityFine:
		return 1
	case QualityRare:
		return 2
	case QualityLegendary:
		return 3
	default:
		return 0
	}
}

func (i *Item) QualityRank() int {
	return i.Quality.Rank()
}

func GetQualityName(q Quality) string {
	switch q {
	case QualityCommon:
		return "普通"
	case QualityFine:
		return "精良"
	case QualityRare:
		return "稀有"
	case QualityLegendary:
		return "传说"
	default:
		return "普通"
	}
}
