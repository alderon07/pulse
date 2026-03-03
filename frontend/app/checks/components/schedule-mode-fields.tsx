"use client";

import { useState } from "react";

type ScheduleMode = "manual" | "auto";

type ScheduleModeFieldsProps = {
  defaultMode: ScheduleMode;
  defaultIntervalSeconds: number;
  defaultGraceSeconds: number;
  labelClassName?: string;
};

export function ScheduleModeFields({
  defaultMode,
  defaultIntervalSeconds,
  defaultGraceSeconds,
  labelClassName = "text-xs uppercase tracking-wide text-slate-500",
}: ScheduleModeFieldsProps) {
  const [mode, setMode] = useState<ScheduleMode>(defaultMode);
  const isAuto = mode === "auto";

  return (
    <>
      <label className={labelClassName}>
        Mode
        <select
          name="schedule_mode"
          value={mode}
          onChange={(event) => setMode(event.target.value as ScheduleMode)}
          className="mt-1.5 w-full rounded border border-slate-800 bg-black px-3 py-2 text-sm text-slate-200 outline-none transition focus:border-green-500/50"
        >
          <option value="auto">auto (learn timing)</option>
          <option value="manual">manual</option>
        </select>
      </label>
      <label className={labelClassName}>
        Interval (seconds)
        <input
          type="number"
          min={1}
          step={1}
          name="expected_interval_seconds"
          defaultValue={defaultIntervalSeconds}
          disabled={isAuto}
          required={!isAuto}
          className="mt-1.5 w-full rounded border border-slate-800 bg-black px-3 py-2 text-sm text-slate-200 outline-none transition focus:border-green-500/50 disabled:cursor-not-allowed disabled:opacity-50"
        />
      </label>
      <label className={labelClassName}>
        Grace (seconds)
        <input
          type="number"
          min={0}
          step={1}
          name="grace_seconds"
          defaultValue={defaultGraceSeconds}
          disabled={isAuto}
          required={!isAuto}
          className="mt-1.5 w-full rounded border border-slate-800 bg-black px-3 py-2 text-sm text-slate-200 outline-none transition focus:border-green-500/50 disabled:cursor-not-allowed disabled:opacity-50"
        />
      </label>
      <div className="text-xs text-slate-600 md:col-span-4 sm:col-span-3">
        {isAuto
          ? "Auto mode learns interval/grace from heartbeat history."
          : "Manual mode uses fixed interval/grace values you set."}
      </div>
    </>
  );
}
