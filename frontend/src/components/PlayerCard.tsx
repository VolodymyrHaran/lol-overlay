import type { Player } from "../types";
import SpellButton from "./SpellButton";
import {
  Card,
  CardContent,
} from "@/components/ui/card";

type Props = {
  player: Player;
  onToggleSpell: (
    gameName: string,
    tagLine: string,
    spell: string
  ) => void;
};

function PlayerCard({ player, onToggleSpell }: Props) {
  return (
    <Card className="mb-4">
      <CardContent className="flex items-center justify-between p-5">
        <div className="flex items-center gap-4">
          <img
            src={`https://ddragon.leagueoflegends.com/cdn/15.24.1/img/champion/${player.champion}.png`}
            alt={player.champion}
            className="h-14 w-14 rounded-lg"
          />

          <div>
            <h3 className="text-lg font-semibold">
              {player.gameName}
              <span className="text-gray-400">#{player.tagLine}</span>
            </h3>

            <p className="mt-1 text-sm text-blue-400">
              {player.champion || "Unknown champion"}
            </p>
          </div>
        </div>

        <div className="flex gap-3">
          {player.spells.map((spell) => (
            <SpellButton
              key={spell.name}
              spell={spell}
              onClick={() =>
                onToggleSpell(
                  player.gameName,
                  player.tagLine,
                  spell.name
                )
              }
            />
          ))}
        </div>
      </CardContent>
    </Card>
  );
}

export default PlayerCard;