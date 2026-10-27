// Branded loading screen for the public site (shown during route transitions and
// suspense) so the user sees the NEXUS mark, never a blank flash.
export default function Loading() {
  return (
    <div className="grid min-h-[60vh] place-items-center bg-[#060b18]">
      <div className="relative grid size-24 place-items-center text-cyan-400">
        <span
          className="absolute inset-0 animate-spin rounded-full border-2 border-transparent"
          style={{ borderTopColor: "currentColor", borderRightColor: "currentColor" }}
        />
        {/* biome-ignore lint/performance/noImgElement: brand asset on a standalone loader */}
        <img src="/brand/mark.webp" alt="" aria-hidden="true" className="size-16 animate-pulse" />
      </div>
    </div>
  );
}
