"use client";

import { type FormEvent, useCallback, useEffect, useState } from "react";

import { BellIcon, CheckCircle2Icon, DownloadIcon, MailCheckIcon, MailWarningIcon, ShieldIcon } from "lucide-react";

import { usePortalI18n } from "./i18n";
import { OrgPolicyPanel } from "./policy";
import { clearPortalToken, downloadExport, type Me, portalApi } from "./portal";
import { SessionsPanel } from "./sessions";
import { SSOPanel } from "./sso";
import { Field, PrimaryButton } from "./ui";

const errText = (ex: unknown, fallback: string) => (ex instanceof Error && ex.message ? ex.message : fallback);

// Per-user 2FA (TOTP) management. Every user manages their own second factor. The
// secret is shown for manual entry plus an otpauth:// link to open in the app —
// never sent to a third-party QR service, so the seed never leaves this origin.
// While the org requires 2FA and the user has not enrolled (`enrolling`), only
// the account sections are shown; `onEnrolled` reloads the portal once they do.
export function SecurityPanel({ me, onEnrolled }: { me: Me; onEnrolled?: () => void }) {
  const { t } = usePortalI18n();
  const enrolling = me.user.mfa_enrollment_required === true;
  const seesPolicy = me.user.role === "owner" || me.user.role === "admin" || me.user.role === "auditor";
  const [enabled, setEnabled] = useState<boolean | null>(null);
  const [setup, setSetup] = useState<{ secret: string; otpauth: string } | null>(null);
  const [code, setCode] = useState("");
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");
  const [note, setNote] = useState("");

  // Email verification + change password.
  const [verified, setVerified] = useState(me.user.email_verified);
  const [resendMsg, setResendMsg] = useState("");
  const [oldPw, setOldPw] = useState("");
  const [newPw, setNewPw] = useState("");
  const [pwMsg, setPwMsg] = useState("");
  const [pwErr, setPwErr] = useState("");
  const [pwBusy, setPwBusy] = useState(false);
  const [notify, setNotify] = useState(me.user.notify_email !== false);
  const [notifyBusy, setNotifyBusy] = useState(false);
  const [notifyMsg, setNotifyMsg] = useState("");
  const [notifyErr, setNotifyErr] = useState("");
  // Bumped to reload the sessions list after a password change ends the others.
  const [sessionsKey, setSessionsKey] = useState(0);

  // Opt in/out of notification mail. Server-side preference: the sender checks it
  // on every email, so it applies on every device at once.
  async function toggleNotify(next: boolean) {
    setNotifyBusy(true);
    setNotifyMsg("");
    setNotifyErr("");
    try {
      await portalApi.setPreferences({ notify_email: next });
      setNotify(next);
      setNotifyMsg(t.sec.notifySaved);
    } catch (ex) {
      setNotifyErr(errText(ex, t.sec.notifyError));
    } finally {
      setNotifyBusy(false);
    }
  }

  async function resendVerification() {
    setResendMsg("");
    try {
      const res = await portalApi.resendVerification();
      if (res.already_verified) {
        setVerified(true);
        setResendMsg(t.sec.alreadyVerified);
      } else {
        setResendMsg(t.sec.resent);
      }
    } catch (ex) {
      setResendMsg(errText(ex, t.sec.resendError));
    }
  }

  // Data & privacy (export / erasure).
  const [exporting, setExporting] = useState(false);
  const [exportErr, setExportErr] = useState("");
  const [delPw, setDelPw] = useState("");
  const [delConfirm, setDelConfirm] = useState(false);
  const [delBusy, setDelBusy] = useState(false);
  const [delErr, setDelErr] = useState("");

  async function exportData() {
    setExporting(true);
    setExportErr("");
    try {
      await downloadExport();
    } catch (ex) {
      setExportErr(errText(ex, t.sec.exportError));
    } finally {
      setExporting(false);
    }
  }

  async function deleteOrg() {
    setDelBusy(true);
    setDelErr("");
    try {
      await portalApi.deleteAccount(delPw);
      clearPortalToken();
      window.location.assign("/portal/login");
    } catch (ex) {
      setDelErr(errText(ex, t.sec.deleteError));
    } finally {
      setDelBusy(false);
    }
  }

  async function changePassword(e: FormEvent) {
    e.preventDefault();
    if (newPw.length < 8) {
      setPwErr(t.passwordMin);
      return;
    }
    setPwBusy(true);
    setPwErr("");
    setPwMsg("");
    try {
      const res = await portalApi.changePassword(oldPw, newPw);
      setOldPw("");
      setNewPw("");
      const ended = res.other_sessions_ended ?? 0;
      setPwMsg(
        ended > 0 ? `${t.sec.passwordUpdated} ${t.sessions.othersEndedAfterPassword(ended)}` : t.sec.passwordUpdated,
      );
      if (ended > 0) setSessionsKey((k) => k + 1);
    } catch (ex) {
      setPwErr(errText(ex, t.sec.passwordError));
    } finally {
      setPwBusy(false);
    }
  }

  const loadStatus = useCallback(async () => {
    try {
      const { enabled } = await portalApi.mfaStatus();
      setEnabled(enabled);
    } catch (ex) {
      // Leave `enabled` unknown: showing "off" when we could not read it would be
      // misleading for a security control.
      setErr(errText(ex, t.sec.loadError));
    }
  }, [t]);

  useEffect(() => {
    void loadStatus();
  }, [loadStatus]);

  async function startSetup() {
    setBusy(true);
    setErr("");
    setNote("");
    try {
      setSetup(await portalApi.mfaSetup());
      setCode("");
    } catch (ex) {
      setErr(errText(ex, t.sec.setupError));
    } finally {
      setBusy(false);
    }
  }

  async function enable() {
    setBusy(true);
    setErr("");
    try {
      await portalApi.mfaEnable(code.trim());
      setSetup(null);
      setCode("");
      setNote(t.sec.enabledNote);
      await loadStatus();
      if (enrolling) onEnrolled?.();
    } catch (ex) {
      setErr(errText(ex, t.sec.invalidCode));
    } finally {
      setBusy(false);
    }
  }

  async function disable() {
    setBusy(true);
    setErr("");
    try {
      await portalApi.mfaDisable(code.trim());
      setCode("");
      setNote(t.sec.disabledNote);
      await loadStatus();
    } catch (ex) {
      setErr(errText(ex, t.sec.invalidCode));
    } finally {
      setBusy(false);
    }
  }

  if (enabled === null) {
    return err ? (
      <p role="alert" className="mt-4 text-sm text-rose-600">
        {err}{" "}
        <button
          type="button"
          onClick={() => {
            setErr("");
            void loadStatus();
          }}
          className="font-semibold underline"
        >
          {t.retry}
        </button>
      </p>
    ) : (
      <p className="mt-4 text-sm text-slate-500">{t.loading}</p>
    );
  }

  return (
    <div className="mt-4 max-w-lg">
      <section className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <h2 className="flex items-center gap-2 text-sm font-semibold uppercase tracking-wide text-cyan-700">
          <ShieldIcon className="size-4" aria-hidden /> {t.sec.mfaTitle}
        </h2>

        {note ? (
          <p role="status" className="mt-3 text-sm text-emerald-700">
            {note}
          </p>
        ) : null}
        {err ? (
          <p role="alert" className="mt-3 text-sm text-rose-600">
            {err}
          </p>
        ) : null}

        {enabled ? (
          <div className="mt-4 space-y-4">
            <p className="flex items-center gap-2 text-sm text-emerald-700">
              <CheckCircle2Icon className="size-4" aria-hidden /> {t.sec.mfaOn}
            </p>
            {me.user.org_requires_mfa ? (
              <p className="text-xs text-slate-500">{t.policy.mfaRequiredByOrg}</p>
            ) : (
              <>
                <p className="text-xs text-slate-500">{t.sec.mfaDisableHint}</p>
                <Field
                  label={t.sec.currentCode}
                  inputMode="numeric"
                  autoComplete="one-time-code"
                  placeholder="123456"
                  value={code}
                  onChange={(e) => setCode(e.target.value)}
                />
                <button
                  type="button"
                  onClick={() => void disable()}
                  disabled={busy || code.trim().length < 6}
                  className="w-full rounded-xl border border-rose-300 px-4 py-2.5 text-sm font-semibold text-rose-600 transition-colors hover:bg-rose-50 disabled:opacity-60"
                >
                  {busy ? t.sec.disabling : t.sec.disable}
                </button>
              </>
            )}
          </div>
        ) : setup ? (
          <div className="mt-4 space-y-4">
            <p className="text-sm text-slate-600">{t.sec.step1}</p>
            <code className="block break-all rounded-lg border border-slate-200 bg-slate-50 px-3 py-2.5 text-sm tracking-widest text-cyan-700">
              {setup.secret}
            </code>
            <a href={setup.otpauth} className="inline-block text-xs text-cyan-700 underline-offset-2 hover:underline">
              {t.sec.openInApp}
            </a>
            <p className="text-sm text-slate-600">{t.sec.step2}</p>
            <Field
              label={t.sec.verifyCode}
              inputMode="numeric"
              autoComplete="one-time-code"
              placeholder="123456"
              value={code}
              onChange={(e) => setCode(e.target.value)}
            />
            <PrimaryButton type="button" onClick={() => void enable()} disabled={busy || code.trim().length < 6}>
              {busy ? t.sec.activating : t.sec.confirmActivate}
            </PrimaryButton>
          </div>
        ) : (
          <div className="mt-4 space-y-4">
            <p className="text-sm text-slate-500">{t.sec.mfaPitch}</p>
            <PrimaryButton type="button" onClick={() => void startSetup()} disabled={busy}>
              {busy ? t.sec.preparing : t.sec.enable}
            </PrimaryButton>
          </div>
        )}
      </section>

      {seesPolicy && !enrolling ? <OrgPolicyPanel me={me} ownMfaEnabled={enabled} /> : null}
      {seesPolicy && !enrolling ? <SSOPanel me={me} /> : null}

      {/* Email verification */}
      <section className="mt-6 rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-cyan-700">{t.email}</h2>
        {verified ? (
          <p className="mt-3 flex items-center gap-2 text-sm text-emerald-700">
            <MailCheckIcon className="size-4" aria-hidden /> {t.sec.verified(me.user.email)}
          </p>
        ) : (
          <div className="mt-3 space-y-3">
            <p className="flex items-center gap-2 text-sm text-amber-700">
              <MailWarningIcon className="size-4" aria-hidden /> {t.sec.unverified(me.user.email)}
            </p>
            <button
              type="button"
              onClick={() => void resendVerification()}
              className="rounded-lg border border-slate-200 px-4 py-2 text-sm text-slate-600 transition-colors hover:border-cyan-400 hover:text-cyan-700"
            >
              {t.sec.resend}
            </button>
            {resendMsg ? (
              <p role="status" className="text-xs text-slate-500">
                {resendMsg}
              </p>
            ) : null}
          </div>
        )}
      </section>

      {/* Notification email preference */}
      <section className="mt-6 rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h2
              id="notify-title"
              className="flex items-center gap-2 text-sm font-semibold uppercase tracking-wide text-cyan-700"
            >
              <BellIcon className="size-4" aria-hidden /> {t.sec.notifyTitle}
            </h2>
            <p id="notify-hint" className="mt-2 text-sm text-slate-600">
              {t.sec.notifyHint}
            </p>
          </div>
          <button
            type="button"
            role="switch"
            aria-checked={notify}
            aria-labelledby="notify-title"
            aria-describedby="notify-hint"
            disabled={notifyBusy}
            onClick={() => void toggleNotify(!notify)}
            className={`relative mt-0.5 inline-flex h-6 w-11 shrink-0 items-center rounded-full transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-400 focus-visible:ring-offset-2 disabled:opacity-60 ${notify ? "bg-cyan-700" : "bg-slate-300"}`}
          >
            <span
              className={`inline-block size-5 rounded-full bg-white shadow transition-transform ${notify ? "translate-x-5" : "translate-x-0.5"}`}
            />
          </button>
        </div>
        {notify && !verified ? <p className="mt-3 text-xs text-amber-700">{t.sec.notifyNeedsVerified}</p> : null}
        <p className="mt-3 text-xs text-slate-500">{t.sec.notifySecurity}</p>
        {notifyMsg ? (
          <p role="status" className="mt-2 text-xs text-emerald-700">
            {notifyMsg}
          </p>
        ) : null}
        {notifyErr ? (
          <p role="alert" className="mt-2 text-xs text-rose-600">
            {notifyErr}
          </p>
        ) : null}
      </section>

      {/* Change password */}
      <section className="mt-6 rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-cyan-700">{t.sec.changePassword}</h2>
        <form onSubmit={changePassword} className="mt-4 space-y-4">
          <Field
            label={t.sec.currentPassword}
            type="password"
            value={oldPw}
            onChange={(e) => setOldPw(e.target.value)}
            autoComplete="current-password"
          />
          <Field
            label={t.sec.newPassword}
            type="password"
            value={newPw}
            onChange={(e) => setNewPw(e.target.value)}
            autoComplete="new-password"
          />
          {pwErr ? (
            <p role="alert" className="text-sm text-rose-600">
              {pwErr}
            </p>
          ) : null}
          {pwMsg ? (
            <p role="status" className="text-sm text-emerald-700">
              {pwMsg}
            </p>
          ) : null}
          <PrimaryButton type="submit" disabled={pwBusy || !oldPw || newPw.length < 8}>
            {pwBusy ? t.sec.saving : t.sec.updatePassword}
          </PrimaryButton>
        </form>
      </section>

      {/* Where this account is signed in */}
      {!enrolling ? <SessionsPanel key={sessionsKey} /> : null}

      {/* Data & privacy */}
      <section
        className={`mt-6 rounded-2xl border border-slate-200 bg-white p-6 shadow-sm ${enrolling ? "hidden" : ""}`}
      >
        <h2 className="text-sm font-semibold uppercase tracking-wide text-cyan-700">{t.sec.privacy}</h2>
        <div className="mt-4 space-y-3">
          <p className="text-sm text-slate-500">{t.sec.exportHint}</p>
          <button
            type="button"
            onClick={() => void exportData()}
            disabled={exporting}
            className="inline-flex items-center gap-2 rounded-lg border border-slate-200 px-4 py-2 text-sm text-slate-600 transition-colors hover:border-cyan-400 hover:text-cyan-700 disabled:opacity-60"
          >
            <DownloadIcon className="size-4" aria-hidden /> {exporting ? t.sec.exporting : t.sec.export}
          </button>
          {exportErr ? (
            <p role="alert" className="text-sm text-rose-600">
              {exportErr}
            </p>
          ) : null}
        </div>

        {me.user.role === "owner" ? (
          <div className="mt-6 border-t border-rose-100 pt-5">
            <h3 className="text-xs font-semibold uppercase tracking-wide text-rose-600">{t.sec.danger}</h3>
            <p className="mt-2 text-sm text-slate-500">{t.sec.dangerHint}</p>
            {!delConfirm ? (
              <button
                type="button"
                onClick={() => setDelConfirm(true)}
                className="mt-3 rounded-lg border border-rose-300 px-4 py-2 text-sm font-semibold text-rose-600 transition-colors hover:bg-rose-50"
              >
                {t.sec.deleteOrg}
              </button>
            ) : (
              <div className="mt-3 space-y-3">
                <Field
                  label={t.sec.confirmWithPassword}
                  type="password"
                  value={delPw}
                  onChange={(e) => setDelPw(e.target.value)}
                  autoComplete="current-password"
                />
                {delErr ? (
                  <p role="alert" className="text-sm text-rose-600">
                    {delErr}
                  </p>
                ) : null}
                <div className="flex flex-wrap gap-2">
                  <button
                    type="button"
                    onClick={() => void deleteOrg()}
                    disabled={delBusy || !delPw}
                    className="rounded-lg bg-rose-600 px-4 py-2 text-sm font-semibold text-white transition-colors hover:bg-rose-700 disabled:opacity-60"
                  >
                    {delBusy ? t.sec.deleting : t.sec.confirmDelete}
                  </button>
                  <button
                    type="button"
                    onClick={() => {
                      setDelConfirm(false);
                      setDelPw("");
                      setDelErr("");
                    }}
                    className="rounded-lg border border-slate-200 px-4 py-2 text-sm text-slate-600 hover:text-slate-900"
                  >
                    {t.cancel}
                  </button>
                </div>
              </div>
            )}
          </div>
        ) : null}
      </section>
    </div>
  );
}
