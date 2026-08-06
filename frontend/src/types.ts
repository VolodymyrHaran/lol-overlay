export type SummonerSpell = {
  name: string;
  isReady: boolean;
  baseCooldown: number;
  remainingCooldown: number;
};

export type Player = {
  champion: string;
  championImage: string;
  gameName: string;
  tagLine: string;
  championId: number;
  summonerSpellHaste: number;
  spells: SummonerSpell[];
};

export type Room = {
  id: string;
  players: Player[];
  lastUpdated: string;
};