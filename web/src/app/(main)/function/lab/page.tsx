"use client";

import { useEffect, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { api } from "@/lib/api";
import { t } from "@/lib/i18n";
import type { LabPrediction, LabRun, LabStatus } from "@/lib/laboratory";

function className(rule: string) {
  const names: Record<string, string> = {
    cleartext: t("lab.class.cleartext"), "missing-hsts": t("lab.class.hsts"),
    "missing-csp": t("lab.class.csp"), "missing-nosniff": t("lab.class.nosniff"),
    "cookie-flags": t("lab.class.cookies"), "login-failures": t("lab.class.logins"),
  };
  return names[rule] ?? rule;
}
function recommendation(rule: string) {
  const messages: Record<string, string> = {
    cleartext: t("lab.fix.cleartext"), "missing-hsts": t("lab.fix.hsts"),
    "missing-csp": t("lab.fix.csp"), "missing-nosniff": t("lab.fix.nosniff"),
    "cookie-flags": t("lab.fix.cookies"), "login-failures": t("lab.fix.logins"),
  };
  return messages[rule] ?? "";
}
const percent = (n: number) => new Intl.NumberFormat("es", { style: "percent", maximumFractionDigits: 1 }).format(n);

export default function LaboratoryPage() {
  const [status, setStatus] = useState<LabStatus | null>(null);
  const [tasks, setTasks] = useState<{ id: string; name?: string; description?: string }[]>([]);
  const [task, setTask] = useState("");
  const [target, setTarget] = useState("");
  const [evidence, setEvidence] = useState("");
  const [runs, setRuns] = useState<LabRun[]>([]);
  const [predictions, setPredictions] = useState<LabPrediction[]>([]);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [busy, setBusy] = useState(false);
  const [loading, setLoading] = useState(true);
  const [historyLoading, setHistoryLoading] = useState(false);
  const selection = useRef(task);
  selection.current = task;

  useEffect(() => {
    let alive = true;
    Promise.all([api.labStatus(), api.tasks()]).then(([state, data]) => {
      if (alive) { setStatus(state); setTasks(data.tasks); }
    }).catch(() => { if (alive) setError(t("lab.loadError")); }).finally(() => { if (alive) setLoading(false); });
    return () => { alive = false; };
  }, []);
  useEffect(() => {
    let alive = true;
    setRuns([]); setPredictions([]); setError(""); setMessage("");
    if (!task) { setHistoryLoading(false); return; }
    setHistoryLoading(true);
    Promise.all([api.labRuns(task), api.labPredictions(task)]).then(([history, ranked]) => {
      if (alive) { setRuns(history); setPredictions(ranked); }
    }).catch(() => { if (alive) setError(t("lab.loadError")); }).finally(() => { if (alive) setHistoryLoading(false); });
    return () => { alive = false; };
  }, [task]);
  async function analyze(event: React.FormEvent) {
    event.preventDefault();
    if (!task || !status?.ready || busy) return;
    const current = task;
    setBusy(true); setError(""); setMessage("");
    try {
      const run = await api.labAnalyze(current, { target: target.trim(), evidence_ids: evidence.split(/[\s,]+/).filter(Boolean) });
      if (selection.current === current) { setRuns((previous) => [run, ...previous].slice(0, 20)); setPredictions(run.predictions); }
    } catch (e) { if (selection.current === current) setError(e instanceof Error ? e.message : t("lab.actionError")); }
    finally { setBusy(false); }
  }
  async function feedback(run: LabRun, rule: string, evidenceID: string, confirmed: boolean) {
    if (!status?.ready || busy) return;
    const current = task;
    setBusy(true); setError(""); setMessage("");
    try {
      await api.labFeedback(current, run.id, { class: rule, evidence_id: evidenceID, confirmed });
      const ranked = await api.labPredictions(current);
      if (selection.current === current) { setPredictions(ranked); setMessage(t("lab.feedbackSaved")); }
    } catch (e) { if (selection.current === current) setError(e instanceof Error ? e.message : t("lab.actionError")); }
    finally { setBusy(false); }
  }
  return (
    <main className="space-y-6 p-4 md:p-6">
      <div><h1 className="text-xl font-semibold">{t("lab.title")}</h1><p className="mt-2 text-sm text-muted-foreground">{t("lab.description")}</p></div>
      <Card className="space-y-2 p-5">
        <h2 className="font-medium">{loading ? t("lab.loading") : status?.ready ? t("lab.ready") : t("lab.blocked")}</h2>
        {status?.blockers.includes("authorization") && <p className="text-sm">{t("lab.authorizationMissing")}</p>}
        {status?.blockers.includes("evidence_source") && <p className="text-sm">{t("lab.evidenceMissing")}</p>}
        <p className="text-sm text-muted-foreground">{t("lab.passive")}</p>
      </Card>
      {error && <p role="alert" className="text-sm text-destructive">{error}</p>}
      {message && <p role="status" className="text-sm">{message}</p>}
      <Card className="space-y-4 p-5">
        <label className="grid gap-2 text-sm" htmlFor="lab-task">{t("lab.task")}
          <select id="lab-task" className="h-10 rounded-md border bg-background px-3" value={task} disabled={loading || busy} onChange={(event) => setTask(event.target.value)}>
            <option value="">{t("lab.selectTask")}</option>
            {tasks.map((item) => <option key={item.id} value={item.id}>{item.name || item.description || item.id}</option>)}
          </select>
        </label>
        <form className="space-y-4" onSubmit={analyze}>
          <label className="grid gap-2 text-sm" htmlFor="lab-target">{t("lab.target")}
            <Input id="lab-target" value={target} onChange={(event) => setTarget(event.target.value)} disabled={!status?.ready || busy} required />
          </label>
          <label className="grid gap-2 text-sm" htmlFor="lab-evidence">{t("lab.evidence")}
            <Input id="lab-evidence" value={evidence} onChange={(event) => setEvidence(event.target.value)} disabled={!status?.ready || busy} required />
          </label>
          <p className="text-xs text-muted-foreground">{t("lab.evidenceHelp")}</p>
          <Button type="submit" disabled={!status?.ready || !task || busy || historyLoading}>{busy ? t("lab.working") : t("lab.analyze")}</Button>
        </form>
      </Card>
      <section className="space-y-3"><h2 className="font-medium">{t("lab.predictions")}</h2>
        <p className="text-sm text-muted-foreground">{t("lab.predictionHelp")}</p>
        {predictions.length > 0 && <div className="overflow-x-auto"><table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="p-2">{t("lab.class")}</th><th className="p-2">{t("lab.samples")}</th><th className="p-2">{t("lab.estimate")}</th><th className="p-2">{t("lab.interval")}</th></tr></thead><tbody>
          {predictions.map((p) => <tr key={p.class} className="border-b"><td className="p-2">{className(p.class)}</td><td className="p-2">{p.samples}</td><td className="p-2">{p.ready ? percent(p.probability) : t("lab.insufficient")}</td><td className="p-2">{p.ready ? `${percent(p.lower)} – ${percent(p.upper)}` : "—"}</td></tr>)}
        </tbody></table></div>}
      </section>
      <section className="space-y-3"><h2 className="font-medium">{t("lab.history")}</h2>
        <p className="text-sm text-muted-foreground">{historyLoading ? t("lab.loading") : !task ? t("lab.selectTask") : runs.length === 0 ? t("lab.empty") : t("lab.latest")}</p>
        {runs.map((run) => <Card key={run.id} className="space-y-3 p-5">
          <div className="flex flex-wrap justify-between gap-2 text-sm"><span className="font-mono">{run.target}</span><time dateTime={run.finished_at}>{new Date(run.finished_at).toLocaleString("es")}</time></div>
          <p className="text-xs text-muted-foreground">{t("lab.run")}: {run.id} · {t("lab.workers")}: {run.workers.length}</p>
          {run.findings.length === 0 && <p className="text-sm">{t("lab.noFindings")}</p>}
          {run.findings.map((finding) => <div key={`${finding.rule}:${finding.evidence_id}`} className="space-y-2 border-t pt-3 text-sm">
            <p className="font-medium">{className(finding.rule)} · {finding.cwe}</p><p>{recommendation(finding.rule)}</p>
            <p className="font-mono text-xs">{finding.evidence_id}</p>
            <div className="flex flex-wrap gap-2"><Button size="sm" variant="outline" disabled={!status?.ready || busy} onClick={() => feedback(run, finding.rule, finding.evidence_id, true)}>{t("lab.confirm")}</Button><Button size="sm" variant="outline" disabled={!status?.ready || busy} onClick={() => feedback(run, finding.rule, finding.evidence_id, false)}>{t("lab.reject")}</Button></div>
          </div>)}
        </Card>)}
      </section>
    </main>
  );
}
