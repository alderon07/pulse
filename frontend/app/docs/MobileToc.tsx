"use client";

import { useState } from "react";
import { List, X } from "lucide-react";
import { tw } from "@/lib/theme";

export function MobileToc({
  items,
}: {
  items: { id: string; label: string }[];
}) {
  const [open, setOpen] = useState(false);

  return (
    <div className="sticky top-0 z-30 bg-black lg:hidden">
      <div className="border-b border-slate-800">
        <button
          type="button"
          onClick={() => setOpen((v) => !v)}
          className="flex w-full items-center justify-between px-4 py-3 text-sm text-slate-400 transition hover:text-green-400"
        >
          <span className="flex items-center gap-2">
            <List size={14} className="text-green-400" />
            On this page
          </span>
          {open ? <X size={14} /> : <span className="text-xs text-slate-600">▼</span>}
        </button>

        {open && (
          <nav className="flex flex-col gap-0.5 border-t border-slate-800 px-4 py-3">
            {items.map((item) => (
              <a
                key={item.id}
                href={`#${item.id}`}
                onClick={() => setOpen(false)}
                className="rounded px-2 py-1.5 text-sm text-slate-400 transition hover:bg-white/[0.03] hover:text-green-400"
              >
                {item.label}
              </a>
            ))}
          </nav>
        )}
      </div>
    </div>
  );
}
