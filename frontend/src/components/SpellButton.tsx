import type { SummonerSpell } from "../types";

type Props = {
  spell: SummonerSpell;
  onClick: () => void;
};

function SpellButton({ spell, onClick }: Props) {
  const buttonClass = spell.isReady
    ? "border-green-500 bg-green-500/10 text-green-400 hover:bg-green-500/20"
    : "border-red-500 bg-red-500/10 text-red-400 hover:bg-red-500/20";

  return (
    <button
      onClick={onClick}
      className={`min-w-24 rounded-lg border px-4 py-3 text-sm font-semibold transition ${buttonClass}`}
    >
      <div>{spell.name}</div>

      <div className="mt-1 text-xs">
        {spell.isReady ? "Ready" : `${spell.remainingCooldown}s`}
      </div>
    </button>
  );
}

export default SpellButton;