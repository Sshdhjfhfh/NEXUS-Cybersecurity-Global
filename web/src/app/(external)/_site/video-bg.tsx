"use client";

import { useEffect, useState } from "react";

// Decorative looping background video. Autoplays muted + inline, loops, and is
// hidden from assistive tech. Falls back to the static poster image when the
// viewer prefers reduced motion (or before hydration).
export function VideoBg({ src, poster, className = "" }: { src: string; poster: string; className?: string }) {
  const [motion, setMotion] = useState(false);
  useEffect(() => {
    setMotion(!window.matchMedia?.("(prefers-reduced-motion: reduce)").matches);
  }, []);

  if (!motion) {
    // biome-ignore lint/performance/noImgElement: static marketing asset
    return <img src={poster} alt="" aria-hidden="true" className={className} />;
  }
  return (
    // biome-ignore lint/a11y/useMediaCaption: decorative background video, no dialogue
    <video
      autoPlay
      muted
      loop
      playsInline
      preload="metadata"
      poster={poster}
      aria-hidden="true"
      tabIndex={-1}
      className={className}
    >
      <source src={src} />
    </video>
  );
}
