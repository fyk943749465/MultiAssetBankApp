import { useMemo, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Checkbox } from "@/components/ui/checkbox";
import { Alert, AlertTitle, AlertDescription } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { buildCodePulseTx, checkCodePulseAction, submitCodePulseTx } from "./api";
import { formatWei, parseEthToWei, payloadToText, shortHash } from "./format";
import type { ActionCheckResponse, CodePulseAction, TxBuildResponse, TxSubmitResponse } from "./types";

type ActionFieldKind = "text" | "textarea" | "address" | "eth" | "bigint" | "boolean" | "multiline_list";

type PrimitiveValue = string | boolean | string[];

type ActionFieldConfig = {
  key: string;
  label: string;
  kind: ActionFieldKind;
  placeholder?: string;
  helpText?: string;
  required?: boolean;
  rows?: number;
};

type ActionFormCardProps = {
  title: string;
  action: CodePulseAction;
  wallet?: string;
  description?: string;
  fields?: ActionFieldConfig[];
  presetParams?: Record<string, PrimitiveValue>;
  proposalId?: number;
  campaignId?: number;
  milestoneIndex?: number;
  submitLabel?: string;
  prepareLabel?: string;
  validate?: (params: Record<string, unknown>) => void;
  onSuccess?: (result: TxSubmitResponse) => void;
};

function defaultValueForKind(kind: ActionFieldKind): PrimitiveValue {
  if (kind === "boolean") return false;
  if (kind === "multiline_list") return [];
  return "";
}

function toFieldRecord(fields: ActionFieldConfig[], presetParams?: Record<string, PrimitiveValue>) {
  const initial: Record<string, PrimitiveValue> = {};
  for (const field of fields) {
    initial[field.key] = defaultValueForKind(field.kind);
  }
  return { ...initial, ...(presetParams ?? {}) };
}

function isNumberish(value: unknown): value is string {
  return typeof value === "string" && /^\d+$/.test(value.trim());
}

export function ActionFormCard({
  title,
  action,
  wallet,
  description,
  fields = [],
  presetParams,
  proposalId,
  campaignId,
  milestoneIndex,
  submitLabel = "发送交易",
  prepareLabel = "预检并构建",
  validate,
  onSuccess,
}: ActionFormCardProps) {
  const [values, setValues] = useState<Record<string, PrimitiveValue>>(() => toFieldRecord(fields, presetParams));
  const [prepareLoading, setPrepareLoading] = useState(false);
  const [submitLoading, setSubmitLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [checkResult, setCheckResult] = useState<ActionCheckResponse | null>(null);
  const [buildResult, setBuildResult] = useState<TxBuildResponse | null>(null);
  const [submitResult, setSubmitResult] = useState<TxSubmitResponse | null>(null);

  const mergedParams = useMemo(() => {
    const params: Record<string, unknown> = {};

    const allEntries = { ...(presetParams ?? {}), ...values };
    for (const field of fields) {
      const rawValue = allEntries[field.key];
      if (rawValue === undefined) continue;
      if (field.kind === "boolean") {
        params[field.key] = Boolean(rawValue);
        continue;
      }

      if (field.kind === "multiline_list") {
        const listSource = Array.isArray(rawValue) ? rawValue : String(rawValue ?? "");
        const list = Array.isArray(listSource)
          ? listSource
          : listSource
              .split(/\r?\n/)
              .map((item) => item.trim())
              .filter(Boolean);
        params[field.key] = list;
        continue;
      }

      const text = String(rawValue ?? "").trim();
      if (!text) continue;

      if (field.kind === "eth") {
        params[field.key] = parseEthToWei(text);
        continue;
      }

      params[field.key] = text;
    }

    for (const [key, value] of Object.entries(presetParams ?? {})) {
      if (params[key] !== undefined) continue;
      params[key] = value;
    }

    return params;
  }, [fields, presetParams, values]);

  function handleTextChange(key: string, nextValue: string) {
    setValues((prev) => ({ ...prev, [key]: nextValue }));
    setBuildResult(null);
    setCheckResult(null);
    setSubmitResult(null);
    setError(null);
  }

  function handleBooleanChange(key: string, nextValue: boolean) {
    setValues((prev) => ({ ...prev, [key]: nextValue }));
    setBuildResult(null);
    setCheckResult(null);
    setSubmitResult(null);
    setError(null);
  }

  function resolveCheckContext() {
    const resolvedProposalId =
      proposalId ?? (isNumberish(String(mergedParams.proposal_id ?? "")) ? Number(mergedParams.proposal_id) : undefined);
    const resolvedCampaignId =
      campaignId ?? (isNumberish(String(mergedParams.campaign_id ?? "")) ? Number(mergedParams.campaign_id) : undefined);
    const resolvedMilestoneIndex =
      milestoneIndex ??
      (isNumberish(String(mergedParams.milestone_index ?? "")) ? Number(mergedParams.milestone_index) : undefined);

    return {
      proposal_id: resolvedProposalId,
      campaign_id: resolvedCampaignId,
      milestone_index: resolvedMilestoneIndex,
    };
  }

  function validateBeforeRequest() {
    if (!wallet) {
      throw new Error("请先连接钱包");
    }

    for (const field of fields) {
      if (!field.required) continue;
      const value = mergedParams[field.key];
      if (field.kind === "boolean") continue;
      if (field.kind === "multiline_list") {
        if (!Array.isArray(value) || value.length === 0) {
          throw new Error(`请填写 ${field.label}`);
        }
        continue;
      }
      if (value === undefined || value === null || String(value).trim() === "") {
        throw new Error(`请填写 ${field.label}`);
      }
    }

    validate?.(mergedParams);
  }

  async function handlePrepare() {
    setPrepareLoading(true);
    setError(null);
    setSubmitResult(null);

    try {
      validateBeforeRequest();
      const checkBody = {
        action,
        wallet: wallet!,
        ...resolveCheckContext(),
        params: mergedParams,
      };
      const check = await checkCodePulseAction(checkBody);
      setCheckResult(check);
      if (!check.allowed) {
        setBuildResult(null);
        return;
      }
      const built = await buildCodePulseTx({
        action,
        wallet: wallet!,
        params: mergedParams,
      });
      setBuildResult(built);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setPrepareLoading(false);
    }
  }

  async function handleSubmit() {
    setSubmitLoading(true);
    setError(null);
    try {
      validateBeforeRequest();
      const build = buildResult
        ? buildResult
        : await buildCodePulseTx({
            action,
            wallet: wallet!,
            params: mergedParams,
          });
      setBuildResult(build);
      if (!build.simulation_ok) {
        throw new Error(build.revert_message || "模拟未通过，无法提交");
      }
      const result = await submitCodePulseTx({
        action,
        wallet: wallet!,
        params: mergedParams,
      });
      setSubmitResult(result);
      onSuccess?.(result);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setSubmitLoading(false);
    }
  }

  return (
    <Card>
      <CardHeader>
        <Badge variant="outline" className="w-fit">{action}</Badge>
        <CardTitle>{title}</CardTitle>
        {description ? <CardDescription>{description}</CardDescription> : null}
      </CardHeader>

      <CardContent className="space-y-4">
        {fields.length > 0 ? (
          <div className="grid gap-4 md:grid-cols-2">
            {fields.map((field) => {
              const value = values[field.key] ?? defaultValueForKind(field.kind);

              if (field.kind === "boolean") {
                return (
                  <label
                    key={field.key}
                    className="flex items-center gap-3 rounded-lg border bg-card/50 px-3 py-3"
                  >
                    <Checkbox
                      checked={Boolean(value)}
                      onCheckedChange={(checked) =>
                        handleBooleanChange(field.key, Boolean(checked))
                      }
                    />
                    <span className="text-sm text-foreground">{field.label}</span>
                  </label>
                );
              }

              if (field.kind === "textarea" || field.kind === "multiline_list") {
                return (
                  <div key={field.key} className="space-y-1 md:col-span-2">
                    <label className="text-sm font-medium text-foreground">
                      {field.label}
                    </label>
                    <Textarea
                      rows={field.rows ?? 4}
                      value={Array.isArray(value) ? value.join("\n") : String(value)}
                      placeholder={field.placeholder}
                      onChange={(e) => handleTextChange(field.key, e.target.value)}
                    />
                    {field.helpText ? (
                      <p className="text-xs text-muted-foreground">{field.helpText}</p>
                    ) : null}
                  </div>
                );
              }

              return (
                <div key={field.key} className="space-y-1">
                  <label className="text-sm font-medium text-foreground">
                    {field.label}
                  </label>
                  <Input
                    value={String(value)}
                    placeholder={field.placeholder}
                    onChange={(e) => handleTextChange(field.key, e.target.value)}
                  />
                  {field.helpText ? (
                    <p className="text-xs text-muted-foreground">{field.helpText}</p>
                  ) : null}
                </div>
              );
            })}
          </div>
        ) : null}

        {!wallet ? (
          <Alert variant="destructive">
            <AlertDescription>
              请先连接钱包后再进行动作预检与交易提交。
            </AlertDescription>
          </Alert>
        ) : null}

        <div className="flex flex-wrap gap-2">
          <Button
            variant="outline"
            disabled={prepareLoading || submitLoading || !wallet}
            onClick={() => void handlePrepare()}
          >
            {prepareLoading ? "构建中…" : prepareLabel}
          </Button>
          <Button
            disabled={
              submitLoading ||
              prepareLoading ||
              !wallet ||
              (buildResult ? !buildResult.simulation_ok : false)
            }
            onClick={() => void handleSubmit()}
          >
            {submitLoading ? "发送中…" : submitLabel}
          </Button>
        </div>

        {error ? (
          <Alert variant="destructive">
            <AlertDescription>
              <pre className="whitespace-pre-wrap break-words font-mono text-sm">
                {error}
              </pre>
            </AlertDescription>
          </Alert>
        ) : null}

        {checkResult ? (
          <div className="space-y-2 text-sm">
            <p className="text-muted-foreground">
              预检结果:{" "}
              <span className={checkResult.allowed ? "text-primary" : "text-warning"}>
                {checkResult.allowed ? "允许执行" : "当前不可执行"}
              </span>
            </p>
            {checkResult.required_role ? (
              <p className="text-muted-foreground">
                所需角色: {checkResult.required_role}
              </p>
            ) : null}
            {checkResult.current_state ? (
              <p className="text-muted-foreground">
                当前状态: {checkResult.current_state}
              </p>
            ) : null}
            {checkResult.reason_message ? (
              <p className="text-warning">{checkResult.reason_message}</p>
            ) : null}
          </div>
        ) : null}

        {buildResult ? (
          <Card>
            <CardHeader>
              <CardDescription>Build Result</CardDescription>
            </CardHeader>
            <CardContent className="space-y-3">
              <div className="grid gap-3 sm:grid-cols-2">
                <div>
                  <p className="text-xs text-muted-foreground">To</p>
                  <p className="mt-1 font-mono text-sm text-foreground">
                    {shortHash(buildResult.to, 10, 8)}
                  </p>
                </div>
                <div>
                  <p className="text-xs text-muted-foreground">Value</p>
                  <p className="mt-1 text-sm text-foreground">
                    {formatWei(buildResult.value)}
                  </p>
                </div>
                <div>
                  <p className="text-xs text-muted-foreground">Simulation</p>
                  <p
                    className={`mt-1 text-sm ${buildResult.simulation_ok ? "text-primary" : "text-warning"}`}
                  >
                    {buildResult.simulation_ok ? "通过" : "未通过"}
                  </p>
                </div>
                <div>
                  <p className="text-xs text-muted-foreground">Gas</p>
                  <p className="mt-1 text-sm text-foreground">
                    {buildResult.gas_estimate ?? "N/A"}
                  </p>
                </div>
              </div>
              {buildResult.revert_message ? (
                <p className="text-sm text-warning">
                  {buildResult.revert_message}
                </p>
              ) : null}
              <details>
                <summary className="cursor-pointer text-sm text-muted-foreground hover:text-foreground">
                  查看 calldata / 返回体
                </summary>
                <pre className="mt-3 whitespace-pre-wrap rounded-lg border bg-muted/50 p-3 font-mono text-xs leading-relaxed text-muted-foreground">
                  {payloadToText(buildResult)}
                </pre>
              </details>
            </CardContent>
          </Card>
        ) : null}

        {submitResult ? (
          <Alert>
            <AlertTitle>交易已提交</AlertTitle>
            <AlertDescription className="font-mono">
              {submitResult.tx_hash}
            </AlertDescription>
          </Alert>
        ) : null}
      </CardContent>
    </Card>
  );
}
