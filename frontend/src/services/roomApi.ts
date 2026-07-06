export async function toggleSpell(
  roomId: string,
  gameName: string,
  tagLine: string,
  spell: string
) {
  await fetch(`http://localhost:8080/rooms/${roomId}/spells/toggle`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      gameName,
      tagLine,
      spell,
    }),
  });
}