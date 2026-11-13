import type { MouseEvent } from "react";
import { startWindowResize, type ResizeEdge } from "../utils/desktop";

const handles: Array<{ edge: ResizeEdge; className: string }> = [
  { edge: "n-resize", className: "resize-handle-north" },
  { edge: "s-resize", className: "resize-handle-south" },
  { edge: "w-resize", className: "resize-handle-west" },
  { edge: "e-resize", className: "resize-handle-east" },
  { edge: "nw-resize", className: "resize-handle-north-west" },
  { edge: "ne-resize", className: "resize-handle-north-east" },
  { edge: "sw-resize", className: "resize-handle-south-west" },
  { edge: "se-resize", className: "resize-handle-south-east" },
];

/**
 * Invisible strips around the frameless window. They provide the resize
 * cursor and hand the native resize gesture to Wails when pressed.
 */
export function ResizeHandles() {
  const handleMouseDown = (event: MouseEvent, edge: ResizeEdge) => {
    if (event.button !== 0) return;
    event.preventDefault();
    event.stopPropagation();
    startWindowResize(edge);
  };

  return (
    <div className="resize-layer" aria-hidden="true">
      {handles.map((handle) => (
        <div
          key={handle.edge}
          className={`resize-handle ${handle.className}`}
          onMouseDown={(event) => handleMouseDown(event, handle.edge)}
        />
      ))}
    </div>
  );
}
