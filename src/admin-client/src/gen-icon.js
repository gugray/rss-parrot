let elmCanvas, elmLink;

export function initGenIcon() {

  // Get elements
  elmCanvas = document.getElementById("cnvIcon");
  elmLink = document.getElementById("lnkSaveIcon");

  // Draw parrot emoji
  const ctx = elmCanvas.getContext("2d");
  ctx.font = "192px Arial";
  ctx.fillText("🦜", 36, 196);

  // Get PNG image and add to link as data URL
  const dataURL = elmCanvas.toDataURL("image/png");
  elmLink.href = dataURL;
  elmLink.download = "favicon-256.png";
}
