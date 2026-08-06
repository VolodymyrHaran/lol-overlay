import type { SummonerSpell } from "../types";

type Props = {
  spell: SummonerSpell;
  onClick: () => void;
};

const DATA_DRAGON_VERSION = "15.24.1";

const spellIconFiles: Record<string, string> = {
  Flash: "SummonerFlash.png",
  Ignite: "SummonerDot.png",
  Heal: "SummonerHeal.png",
  Ghost: "SummonerHaste.png",
  Exhaust: "SummonerExhaust.png",
  Barrier: "SummonerBarrier.png",
  Cleanse: "SummonerBoost.png",
  Teleport: "SummonerTeleport.png",
  Smite: "SummonerSmite.png",
};

function formatCooldown(seconds: number): string {
  const safeSeconds = Math.max(0, Math.floor(seconds));
  const minutes = Math.floor(safeSeconds / 60);
  const remainingSeconds = safeSeconds % 60;

  return `${minutes}:${remainingSeconds
    .toString()
    .padStart(2, "0")}`;
}

function getSpellIconURL(spellName: string): string | null {
  const fileName = spellIconFiles[spellName];

  if (!fileName) {
    return null;
  }

  return (
    `https://ddragon.leagueoflegends.com/cdn/` +
    `${DATA_DRAGON_VERSION}/img/spell/${fileName}`
  );
}

function SpellButton({
  spell,
  onClick,
}: Props) {
  const iconURL = getSpellIconURL(spell.name);

  const buttonClass = spell.isReady
    ? [
        "border-emerald-500/70",
        "bg-emerald-500/10",
        "text-emerald-300",
        "hover:bg-emerald-500/20",
      ].join(" ")
    : [
        "border-red-500/70",
        "bg-red-500/10",
        "text-red-300",
        "hover:bg-red-500/20",
      ].join(" ");

  return (
    <button
      type="button"
      onClick={onClick}
      aria-label={`${spell.name} ${
        spell.isReady
          ? "ready"
          : `${spell.remainingCooldown} seconds remaining`
      }`}
      className={[
        "group relative min-w-24 overflow-hidden",
        "rounded-xl border px-3 py-3",
        "transition duration-150",
        "active:scale-95",
        buttonClass,
      ].join(" ")}
    >
      <div className="flex flex-col items-center gap-2">
        <div className="relative h-11 w-11 overflow-hidden rounded-lg bg-zinc-800">
          {iconURL ? (
            <img
              src={iconURL}
              alt=""
              className={[
                "h-full w-full object-cover",
                "transition",
                spell.isReady
                  ? "opacity-100"
                  : "opacity-65 grayscale-[25%]",
              ].join(" ")}
            />
          ) : (
            <div className="flex h-full w-full items-center justify-center text-xs text-zinc-500">
              ?
            </div>
          )}

          {!spell.isReady && (
            <div className="absolute inset-0 bg-black/25" />
          )}
        </div>

        <div className="text-sm font-semibold">
          {spell.name}
        </div>

        <div
          className={[
            "min-h-5 text-xs font-bold tabular-nums",
            spell.isReady
              ? "text-emerald-400"
              : "text-red-300",
          ].join(" ")}
        >
          {spell.isReady
            ? "Ready"
            : formatCooldown(spell.remainingCooldown)}
        </div>
      </div>
    </button>
  );
}

export default SpellButton;