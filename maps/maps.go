package maps

type Map struct {
	ID   string
	Name string
}

var (
	Start = Map{"start", "Entrance"}
	E1M1  = Map{"e1m1", "Slipgate Complex"}
	E1M2  = Map{"e1m2", "Castle of the Damned"}
	E1M3  = Map{"e1m3", "The Necropolis"}
	E1M4  = Map{"e1m4", "The Grisly Grotto"}
	E1M5  = Map{"e1m5", "Gloom Keep"}
	E1M6  = Map{"e1m6", "The Door To Chthon"}
	E1M7  = Map{"e1m7", "The House of Chthon"}
	E1M8  = Map{"e1m8", "Ziggurat Vertigo"}
	E2M1  = Map{"e2m1", "The Installation"}
	E2M2  = Map{"e2m2", "Ogre Citadel"}
	E2M3  = Map{"e2m3", "Crypt of Decay"}
	E2M4  = Map{"e2m4", "The Ebon Fortress"}
	E2M5  = Map{"e2m5", "The Wizard's Manse"}
	E2M6  = Map{"e2m6", "The Dismal Oubliette"}
	E2M7  = Map{"e2m7", "Underearth"}
	E3M1  = Map{"e3m1", "Termination Central"}
	E3M2  = Map{"e3m2", "The Vaults of Zin"}
	E3M3  = Map{"e3m3", "The Tomb of Terror"}
	E3M4  = Map{"e3m4", "Satan's Dark Delight"}
	E3M5  = Map{"e3m5", "Wind Tunnels"}
	E3M6  = Map{"e3m6", "Chambers of Torment"}
	E3M7  = Map{"e3m7", "The Haunted Halls"}
	E4M1  = Map{"e4m1", "The Sewage System"}
	E4M2  = Map{"e4m2", "The Tower of Despair"}
	E4M3  = Map{"e4m3", "The Elder God Shrine"}
	E4M4  = Map{"e4m4", "The Palace of Hate"}
	E4M5  = Map{"e4m5", "Hell's Atrium"}
	E4M6  = Map{"e4m6", "The Pain Maze"}
	E4M7  = Map{"e4m7", "Azure Agony"}
	E4M8  = Map{"e4m8", "The Nameless City"}
	End   = Map{"end", "Shub-Niggurath's Pit"}

	DM1 = Map{"dm1", "Place of Two Deaths"}
	DM2 = Map{"dm2", "Claustrophobopolis"}
	DM3 = Map{"dm3", "The Abandoned Base"}
	DM4 = Map{"dm4", "The Bad Place"}
	DM5 = Map{"dm5", "The Cistern"}
	DM6 = Map{"dm6", "The Dark Zone"}

	HipStart = Map{"start", "Command HQ"}
	Hip1M1   = Map{"hip1m1", "The Pumping Station"}
	Hip1M2   = Map{"hip1m2", "Storage Facility"}
	Hip1M3   = Map{"hip1m3", "The Lost Mine"}
	Hip1M4   = Map{"hip1m4", "Research Facility"}
	Hip1M5   = Map{"hip1m5", "Military Complex"}
	Hip2M1   = Map{"hip2m1", "Ancient Realms"}
	Hip2M2   = Map{"hip2m2", "The Black Cathedral"}
	Hip2M3   = Map{"hip2m3", "The Catacombs"}
	Hip2M4   = Map{"hip2m4", "The Crypt"}
	Hip2M5   = Map{"hip2m5", "Mortum's Keep"}
	Hip2M6   = Map{"hip2m6", "The Gremlin's Domain"}
	Hip3M1   = Map{"hip3m1", "Tur Torment"}
	Hip3M2   = Map{"hip3m2", "Pandemonium"}
	Hip3M3   = Map{"hip3m3", "Limbo"}
	Hip3M4   = Map{"hip3m4", "The Gauntlet"}
	HipEnd   = Map{"hipend", "Armagon's Lair"}

	HipDM1 = Map{"hipdm1", "The Edge of Oblivion"}

	RStart = Map{"start", "Split Decision"}
	R1M1   = Map{"r1m1", "Deviant's Domain"}
	R1M2   = Map{"r1m2", "Dread Portal"}
	R1M3   = Map{"r1m3", "Judgement Call"}
	R1M4   = Map{"r1m4", "Cave of Death"}
	R1M5   = Map{"r1m5", "Towers of Wrath"}
	R1M6   = Map{"r1m6", "Temple of Pain"}
	R1M7   = Map{"r1m7", "Tomb of the Overlord"}
	R2M1   = Map{"r2m1", "Tempus Fugit"}
	R2M2   = Map{"r2m2", "Elemental Fury I"}
	R2M3   = Map{"r2m3", "Elemental Fury II"}
	R2M4   = Map{"r2m4", "Curse of Osiris"}
	R2M5   = Map{"r2m5", "Wizard's Keep"}
	R2M6   = Map{"r2m6", "Blood Sacrifice"}
	R2M7   = Map{"r2m7", "Last Bastion"}
	R2M8   = Map{"r2m8", "Source of Evil"}

	CTF1 = Map{"ctf1", "Division of Change"}
)

type Episode struct {
	Name string
	Maps []Map
}

var (
	ES  = Episode{"Welcome to Quake", []Map{Start}}
	E1  = Episode{"Doomed Dimension", []Map{E1M1, E1M2, E1M3, E1M4, E1M5, E1M6, E1M7, E1M8}}
	E2  = Episode{"Realm of Black Magic", []Map{E2M1, E2M2, E2M3, E2M4, E2M5, E2M6, E2M7}}
	E3  = Episode{"Netherworld", []Map{E3M1, E3M2, E3M3, E3M4, E3M5, E3M6, E3M7}}
	E4  = Episode{"The Elder World", []Map{E4M1, E4M2, E4M3, E4M4, E4M5, E4M6, E4M7, E4M8}}
	EE  = Episode{"Final Level", []Map{End}}
	EDM = Episode{"Deathmatch Arena", []Map{DM1, DM2, DM3, DM4, DM5, DM6}}
	HS  = Episode{"Scourge of Armagon", []Map{HipStart}}
	H1  = Episode{"Fortress of the Dead", []Map{Hip1M1, Hip1M2, Hip1M3, Hip1M4, Hip1M5}}
	H2  = Episode{"Dominion of Darkness", []Map{Hip2M1, Hip2M2, Hip2M3, Hip2M4, Hip2M5, Hip2M6}}
	H3  = Episode{"The Rift", []Map{Hip3M1, Hip3M2, Hip3M3, Hip3M4}}
	HE  = Episode{"Final Level", []Map{HipEnd}}
	HDM = Episode{"Deathmatch Arena", []Map{HipDM1}}

	RS  = Episode{"Introduction", []Map{RStart}}
	R1  = Episode{"Hell's Fortress", []Map{R1M1, R1M2, R1M3, R1M4, R1M5, R1M5, R1M6, R1M7}}
	R2  = Episode{"Corridors of Time", []Map{R2M1, R2M2, R2M3, R2M4, R2M5, R2M6, R2M7, R2M8}}
	RDM = Episode{"Deathmatch Arena", []Map{CTF1}}
)

func Base() []Episode {
	return []Episode{ES, E1, E2, E3, E4, EE, EDM}
}
func Hipnotic() []Episode {
	return []Episode{HS, H1, H2, H3, HE, HDM}
}
func Rogue() []Episode {
	return []Episode{RS, R1, R2, RDM}
}
