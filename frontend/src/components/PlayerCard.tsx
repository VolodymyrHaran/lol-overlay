import type { Player } from "../types";
import SpellButton from "./SpellButton";
import { useEffect, useState } from "react";


import {
  Card,
  CardContent,
} from "@/components/ui/card";

type Props = {
  player: Player;
  onToggleSpell: (
    gameName: string,
    tagLine: string,
    spell: string,
  ) => void;
};


function PlayerCard({
  player,
  onToggleSpell,
}: Props) {
  const [imageLoaded, setImageLoaded] = useState(false);

  useEffect(() => {
    setImageLoaded(false);
  }, [player.championImage]);


  return (
    <Card className="mb-4">
      <CardContent className="flex items-center justify-between p-5">
        <div className="relative h-14 w-14 shrink-0 overflow-hidden rounded-lg bg-zinc-800">
          {!imageLoaded && (
            <div className="absolute inset-0 animate-pulse bg-zinc-700" />
          )}

          {player.championImage && (
            <img
              key={player.championImage}
              src={player.championImage}
              alt={player.champion}
              className={`h-full w-full object-cover transition-opacity duration-200 ${
                imageLoaded ? "opacity-100" : "opacity-0"
              }`}
              onLoad={() => setImageLoaded(true)}
              onError={() => setImageLoaded(false)}
            />
          )}
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
                  spell.name,
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