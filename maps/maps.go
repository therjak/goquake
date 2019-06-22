package maps

type Map struct {
	ID   string
	Name string
}

var (
	start = Map{"start", "Entrance"}
	e1m1  = Map{"e1m1", "Slipgate Complex"}
	e1m2  = Map{"e1m2", "Castle of the Damned"}
	e1m3  = Map{"e1m3", "The Necropolis"}
	e1m4  = Map{"e1m4", "The Grisly Grotto"}
	e1m5  = Map{"e1m5", "Gloom Keep"}
	e1m6  = Map{"e1m6", "The Door To Chthon"}
	e1m7  = Map{"e1m7", "The House of Chthon"}
	e1m8  = Map{"e1m8", "Ziggurat Vertigo"}
	e2m1  = Map{"e2m1", "The Installation"}
	e2m2  = Map{"e2m2", "Ogre Citadel"}
	e2m3  = Map{"e2m3", "Crypt of Decay"}
	e2m4  = Map{"e2m4", "The Ebon Fortress"}
	e2m5  = Map{"e2m5", "The Wizard's Manse"}
	e2m6  = Map{"e2m6", "The Dismal Oubliette"}
	e2m7  = Map{"e2m7", "Underearth"}
	e3m1  = Map{"e3m1", "Termination Central"}
	e3m2  = Map{"e3m2", "The Vaults of Zin"}
	e3m3  = Map{"e3m3", "The Tomb of Terror"}
	e3m4  = Map{"e3m4", "Satan's Dark Delight"}
	e3m5  = Map{"e3m5", "Wind Tunnels"}
	e3m6  = Map{"e3m6", "Chambers of Torment"}
	e3m7  = Map{"e3m7", "The Haunted Halls"}
	e4m1  = Map{"e4m1", "The Sewage System"}
	e4m2  = Map{"e4m2", "The Tower of Despair"}
	e4m3  = Map{"e4m3", "The Elder God Shrine"}
	e4m4  = Map{"e4m4", "The Palace of Hate"}
	e4m5  = Map{"e4m5", "Hell's Atrium"}
	e4m6  = Map{"e4m6", "The Pain Maze"}
	e4m7  = Map{"e4m7", "Azure Agony"}
	e4m8  = Map{"e4m8", "The Nameless City"}
	end   = Map{"end", "Shub-Niggurath's Pit"}

	dm1 = Map{"dm1", "Place of Two Deaths"}
	dm2 = Map{"dm2", "Claustrophobopolis"}
	dm3 = Map{"dm3", "The Abandoned Base"}
	dm4 = Map{"dm4", "The Bad Place"}
	dm5 = Map{"dm5", "The Cistern"}
	dm6 = Map{"dm6", "The Dark Zone"}

	hipStart = Map{"start", "Command HQ"}
	hip1m1   = Map{"hip1m1", "The Pumping Station"}
	hip1m2   = Map{"hip1m2", "Storage Facility"}
	hip1m3   = Map{"hip1m3", "The Lost Mine"}
	hip1m4   = Map{"hip1m4", "Research Facility"}
	hip1m5   = Map{"hip1m5", "Military Complex"}
	hip2m1   = Map{"hip2m1", "Ancient Realms"}
	hip2m2   = Map{"hip2m2", "The Black Cathedral"}
	hip2m3   = Map{"hip2m3", "The Catacombs"}
	hip2m4   = Map{"hip2m4", "The Crypt"}
	hip2m5   = Map{"hip2m5", "Mortum's Keep"}
	hip2m6   = Map{"hip2m6", "The Gremlin's Domain"}
	hip3m1   = Map{"hip3m1", "Tur Torment"}
	hip3m2   = Map{"hip3m2", "Pandemonium"}
	hip3m3   = Map{"hip3m3", "Limbo"}
	hip3m4   = Map{"hip3m4", "The Gauntlet"}
	hipEnd   = Map{"hipend", "Armagon's Lair"}

	hipDm1 = Map{"hipdm1", "The Edge of Oblivion"}

	rStart = Map{"start", "Split Decision"}
	r1m1   = Map{"r1m1", "Deviant's Domain"}
	r1m2   = Map{"r1m2", "Dread Portal"}
	r1m3   = Map{"r1m3", "Judgement Call"}
	r1m4   = Map{"r1m4", "Cave of Death"}
	r1m5   = Map{"r1m5", "Towers of Wrath"}
	r1m6   = Map{"r1m6", "Temple of Pain"}
	r1m7   = Map{"r1m7", "Tomb of the Overlord"}
	r2m1   = Map{"r2m1", "Tempus Fugit"}
	r2m2   = Map{"r2m2", "Elemental Fury I"}
	r2m3   = Map{"r2m3", "Elemental Fury II"}
	r2m4   = Map{"r2m4", "Curse of Osiris"}
	r2m5   = Map{"r2m5", "Wizard's Keep"}
	r2m6   = Map{"r2m6", "Blood Sacrifice"}
	r2m7   = Map{"r2m7", "Last Bastion"}
	r2m8   = Map{"r2m8", "Source of Evil"}

	ctf1 = Map{"ctf1", "Division of Change"}
)

type Episode struct {
	Name string
	Maps []Map
}

var (
	es  = Episode{"Welcome to Quake", []Map{start}}
	e1  = Episode{"Doomed Dimension", []Map{e1m1, e1m2, e1m3, e1m4, e1m5, e1m6, e1m7, e1m8}}
	e2  = Episode{"Realm of Black Magic", []Map{e2m1, e2m2, e2m3, e2m4, e2m5, e2m6, e2m7}}
	e3  = Episode{"Netherworld", []Map{e3m1, e3m2, e3m3, e3m4, e3m5, e3m6, e3m7}}
	e4  = Episode{"The Elder World", []Map{e4m1, e4m2, e4m3, e4m4, e4m5, e4m6, e4m7, e4m8}}
	ee  = Episode{"Final Level", []Map{end}}
	edm = Episode{"Deathmatch Arena", []Map{dm1, dm2, dm3, dm4, dm5, dm6}}
	hs  = Episode{"Scourge of Armagon", []Map{hipStart}}
	h1  = Episode{"Fortress of the Dead", []Map{hip1m1, hip1m2, hip1m3, hip1m4, hip1m5}}
	h2  = Episode{"Dominion of Darkness", []Map{hip2m1, hip2m2, hip2m3, hip2m4, hip2m5, hip2m6}}
	h3  = Episode{"The Rift", []Map{hip3m1, hip3m2, hip3m3, hip3m4}}
	he  = Episode{"Final Level", []Map{hipEnd}}
	hdm = Episode{"Deathmatch Arena", []Map{hipDm1}}

	rs  = Episode{"Introduction", []Map{rStart}}
	r1  = Episode{"Hell's Fortress", []Map{r1m1, r1m2, r1m3, r1m4, r1m5, r1m5, r1m6, r1m7}}
	r2  = Episode{"Corridors of Time", []Map{r2m1, r2m2, r2m3, r2m4, r2m5, r2m6, r2m7, r2m8}}
	rdm = Episode{"Deathmatch Arena", []Map{ctf1}}
)

func Base() []Episode {
	return []Episode{es, e1, e2, e3, e4, ee, edm}
}
func Hipnotic() []Episode {
	return []Episode{hs, h1, h2, h3, he, hdm}
}
func Rogue() []Episode {
	return []Episode{rs, r1, r2, rdm}
}
