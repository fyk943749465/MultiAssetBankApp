import { useMemo, useRef, useState } from "react";
import { useChainId, useSendTransaction, useSwitchChain } from "wagmi";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Checkbox } from "@/components/ui/checkbox";
import { Alert, AlertTitle, AlertDescription } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { buildCodePulseTx, checkCodePulseAction } from "./api";
import { formatDuration, formatWei, parseEthToWei, payloadToText, shortHash } from "./format";
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
  const chainId = useChainId();
  const { switchChainAsync } = useSwitchChain();
  const { sendTransactionAsync, isPending: walletSendPending } = useSendTransaction();

  const [values, setValues] = useState<Record<string, PrimitiveValue>>(() => toFieldRecord(fields, presetParams));
  const [prepareLoading, setPrepareLoading] = useState(false);
  const [submitLoading, setSubmitLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [checkResult, setCheckResult] = useState<ActionCheckResponse | null>(null);
  const [buildResult, setBuildResult] = useState<TxBuildResponse | null>(null);
  const [submitResult, setSubmitResult] = useState<TxSubmitResponse | null>(null);
  const prepareInFlightRef = useRef(false);
  const submitInFlightRef = useRef(false);

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
        try {
          params[field.key] = parseEthToWei(text);
        } catch {
          // 输入过程中的中间态（如 "0."）会解析失败；不在 render 中抛错，避免整页白屏。提交时由必填校验捕获。
        }
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
    if (prepareInFlightRef.current) return;
    prepareInFlightRef.current = true;
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
      prepareInFlightRef.current = false;
      setPrepareLoading(false);
    }
  }

  async function handleSubmit() {
    if (submitInFlightRef.current) return;
    if (submitResult) {
      setError("该交易已提交成功，请勿重复发送。如需再次操作请刷新页面。");
      return;
    }
    submitInFlightRef.current = true;
    setSubmitLoading(true);
    setError(null);
    try {
      validateBeforeRequest();
      const build = await buildCodePulseTx({
        action,
        wallet: wallet!,
        params: mergedParams,
      });
      setBuildResult(build);
      if (!build.simulation_ok) {
        throw new Error(build.revert_message || "模拟未通过，无法提交");
      }

      const targetChain = build.chain_id;
      if (targetChain != null && chainId !== targetChain) {
        if (!switchChainAsync) {
          throw new Error(`请先在钱包中切换到 chainId=${targetChain} 的网络`);
        }
        await switchChainAsync({ chainId: targetChain });
      }

      const valueWei =
        build.value === "" || build.value === "0" || build.value === undefined ? 0n : BigInt(build.value);

      const hash = await sendTransactionAsync({
        to: build.to as `0x${string}`,
        data: build.data as `0x${string}`,
        value: valueWei,
        gas: build.gas_estimate != null ? BigInt(build.gas_estimate) : undefined,
      });

      const result: TxSubmitResponse = {
        tx_hash: hash,
        action,
        from: wallet!,
        tx_submit_mode: "wallet_sign",
        request_wallet: wallet!,
      };
      setSubmitResult(result);
      onSuccess?.(result);
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err);
      if (/user rejected|denied|cancel/i.test(msg)) {
        setError("已在钱包中取消签名或发送");
      } else {
        setError(msg);
      }
    } finally {
      submitInFlightRef.current = false;
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

        {wallet ? (
          <Alert>
            <AlertTitle>钱包签名发送</AlertTitle>
            <AlertDescription className="text-sm text-muted-foreground">
              点击「{submitLabel}」将在钱包中请求确认；链上交易的 <code className="rounded bg-muted px-1">from</code>{" "}
              为当前连接地址，合约中 <code className="rounded bg-muted px-1">msg.sender</code>（如提案发起人）与之一致。
            </AlertDescription>
          </Alert>
        ) : null}

        <div className="flex flex-wrap gap-2">
          <Button
            variant="outline"
            disabled={prepareLoading || submitLoading || walletSendPending || !wallet}
            onClick={() => void handlePrepare()}
          >
            {prepareLoading ? "构建中…" : prepareLabel}
          </Button>
          <Button
            disabled={
              submitLoading ||
              walletSendPending ||
              prepareLoading ||
              !wallet ||
              !!submitResult ||
              (buildResult ? !buildResult.simulation_ok : false)
            }
            onClick={() => void handleSubmit()}
          >
            {submitLoading || walletSendPending
              ? "钱包确认中…"
              : submitResult
                ? "已提交"
                : submitLabel}
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
                {buildResult.tx_submit_signer ? (
                  <div className="sm:col-span-2">
                    <p className="text-xs text-muted-foreground">链上发送方（当前钱包 / msg.sender）</p>
                    <p className="mt-1 font-mono text-sm text-foreground break-all">
                      {buildResult.tx_submit_signer}
                    </p>
                  </div>
                ) : null}
                {buildResult.target_wei_packed ? (
                  <div className="sm:col-span-2">
                    <p className="text-xs text-muted-foreground">打包目标（wei）</p>
                    <p className="mt-1 font-mono text-sm text-foreground break-all">
                      {buildResult.target_wei_packed}
                    </p>
                  </div>
                ) : null}
                {buildResult.duration_seconds_packed ? (
                  <div className="sm:col-span-2">
                    <p className="text-xs text-muted-foreground">打包众筹时长（秒）</p>
                    <p className="mt-1 font-mono text-sm text-foreground">
                      {buildResult.duration_seconds_packed}{" "}
                      <span className="text-muted-foreground">
                        （约 {formatDuration(Number(buildResult.duration_seconds_packed))}）
                      </span>
                    </p>
                  </div>
                ) : null}
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
            <AlertDescription className="space-y-2">
              <p className="font-mono break-all">{submitResult.tx_hash}</p>
              {submitResult.from ? (
                <p className="text-sm text-muted-foreground">
                  链上 <code className="rounded bg-muted px-1">from</code>:{" "}
                  <span className="font-mono text-foreground">{submitResult.from}</span>
                </p>
              ) : null}
            </AlertDescription>
          </Alert>
        ) : null}
      </CardContent>
    </Card>
  );
}
