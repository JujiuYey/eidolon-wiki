import { reactive } from "vue";

import type { PaperSize } from "../_shared/const";
import {
  DEFAULT_MARGINS,
  DEFAULT_PAPER_SIZE,
  parsePaperSize,
  ZOOM_DEFAULT,
  ZOOM_MAX,
  ZOOM_MIN,
  ZOOM_STEP,
} from "../_shared/const";

export interface PageMargins {
  top: number;
  bottom: number;
  left: number;
  right: number;
}

export interface EditorPageSetup {
  zoom: number;
  margins: PageMargins;
  paperSize: PaperSize;
  zoomIn: () => void;
  zoomOut: () => void;
  zoomReset: () => void;
  setZoom: (value: number) => void;
  setMargins: (margins: Partial<PageMargins>) => void;
  resetMargins: () => void;
  setPaperSize: (value: string) => void;
}

function clampZoom(v: number) {
  return Math.round(Math.min(ZOOM_MAX, Math.max(ZOOM_MIN, v)) * 100) / 100;
}

export function useEditorPageSetup(): EditorPageSetup {
  const setup = reactive<EditorPageSetup>({
    zoom: ZOOM_DEFAULT,
    margins: { ...DEFAULT_MARGINS },
    paperSize: { ...DEFAULT_PAPER_SIZE },
    zoomIn: () => {
      setup.zoom = clampZoom(setup.zoom + ZOOM_STEP);
    },
    zoomOut: () => {
      setup.zoom = clampZoom(setup.zoom - ZOOM_STEP);
    },
    zoomReset: () => {
      setup.zoom = ZOOM_DEFAULT;
    },
    setZoom: (value) => {
      setup.zoom = clampZoom(value);
    },
    setMargins: (partial) => {
      Object.assign(setup.margins, partial);
    },
    resetMargins: () => {
      Object.assign(setup.margins, DEFAULT_MARGINS);
    },
    setPaperSize: (value) => {
      Object.assign(setup.paperSize, parsePaperSize(value));
    },
  });

  return setup;
}
