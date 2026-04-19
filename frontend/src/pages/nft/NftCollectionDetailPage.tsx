import { useCallback, useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { fetchNftCollectionById, fetchNftCollectionTokens, type NftCollectionDbRow, type NftTokenRow } from "@/features/nft/api";
import { shortHash } from "@/features/codepulse/format";

function explorerAddress(chainId: number, addr: string): string {
  if (chainId === 11_155_111) return `https://sepolia.etherscan.io/address/${addr}`;
  return `https://etherscan.io/address/${addr}`;
}

function explorerTx(chainId: number, hash: string): string {
  if (chainId === 11_155_111) return `https://sepolia.etherscan.io/tx/${hash}`;
  return `https://etherscan.io/tx/${hash}`;
}

export function NftCollectionDetailPage() {
  const { collectionId } = useParams<{ collectionId: string }>();
  const idNum = collectionId ? Number(collectionId) : Number.NaN;

  const [collection, setCollection] = useState<NftCollectionDbRow | null>(null);
  const [dataSource, setDataSource] = useState<string>("database");
  const [tokens, setTokens] = useState<NftTokenRow[]>([]);
  const [tokenTotal, setTokenTotal] = useState(0);
  const [tokenPage, setTokenPage] = useState(1);
  const tokenPageSize = 30;
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState<string | null>(null);

  const loadCollection = useCallback(async () => {
    if (!collectionId || !Number.isFinite(idNum) || idNum <= 0) {
      setErr("无效的合集 ID");
      setCollection(null);
      return;
    }
    setErr(null);
    try {
      const r = await fetchNftCollectionById(collectionId);
      setDataSource(r.data_source ?? "database");
      setCollection(r.collection);
    } catch (e) {
      setCollection(null);
      setErr(e instanceof Error ? e.message : String(e));
    }
  }, [collectionId, idNum]);

  const loadTokens = useCallback(async () => {
    if (!collectionId || !Number.isFinite(idNum) || idNum <= 0) return;
    try {
      const r = await fetchNftCollectionTokens(collectionId, { page: tokenPage, page_size: tokenPageSize });
      setTokens(r.tokens ?? []);
      setTokenTotal(r.total ?? 0);
    } catch {
      setTokens([]);
      setTokenTotal(0);
    }
  }, [collectionId, idNum, tokenPage]);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      setLoading(true);
      await loadCollection();
      if (cancelled) return;
      setLoading(false);
    })();
    return () => {
      cancelled = true;
    };
  }, [loadCollection]);

  useEffect(() => {
    void loadTokens();
  }, [loadTokens]);

  if (!collectionId || !Number.isFinite(idNum) || idNum <= 0) {
    return (
      <Alert variant="destructive">
        <AlertTitle>参数错误</AlertTitle>
        <AlertDescription>请从概览页的库内合集进入。</AlertDescription>
      </Alert>
    );
  }

  if (loading) {
    return <p className="text-sm text-muted-foreground">加载中…</p>;
  }

  if (err || !collection) {
    return (
      <div className="space-y-4">
        <Link
          to="/nft"
          className="inline-flex h-7 items-center rounded-[min(var(--radius-md),12px)] border border-border bg-background px-2.5 text-[0.8rem] font-medium hover:bg-muted"
        >
          ← 返回 NFT 概览
        </Link>
        <Alert variant="destructive">
          <AlertTitle>无法加载合集</AlertTitle>
          <AlertDescription>{err ?? "未找到"}</AlertDescription>
        </Alert>
      </div>
    );
  }

  const cid = collection.chain_id;

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center gap-3">
        <Link
          to="/nft"
          className="inline-flex h-7 items-center rounded-[min(var(--radius-md),12px)] border border-border bg-background px-2.5 text-[0.8rem] font-medium hover:bg-muted"
        >
          ← 返回概览
        </Link>
        <Link
          to={`/nft/collections/${collection.contract_address}/mint`}
          className="inline-flex h-7 items-center rounded-[min(var(--radius-md),12px)] border border-primary/30 bg-primary/10 px-2.5 text-[0.8rem] font-medium text-primary hover:bg-primary/15"
        >
          铸造 NFT
        </Link>
        <Badge variant="secondary" className="font-mono text-[10px]">
          data_source: {dataSource}
        </Badge>
      </div>

      <Card className="border-white/10 bg-card/40">
        <CardHeader>
          <CardTitle className="text-lg">
            {collection.collection_name?.trim() || `合集 #${collection.id}`}
          </CardTitle>
          <CardDescription className="font-mono text-xs">
            symbol: {collection.collection_symbol ?? "—"} · chain_id {cid}
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-3 text-sm">
          <div>
            <span className="text-muted-foreground">合约地址 </span>
            <a
              href={explorerAddress(cid, collection.contract_address)}
              target="_blank"
              rel="noreferrer"
              className="break-all font-mono text-primary underline-offset-4 hover:underline"
            >
              {collection.contract_address}
            </a>
          </div>
          <div>
            <span className="text-muted-foreground">创建者 </span>
            <a
              href={explorerAddress(cid, collection.creator_address)}
              target="_blank"
              rel="noreferrer"
              className="break-all font-mono text-primary underline-offset-4 hover:underline"
            >
              {shortHash(collection.creator_address, 8, 6)}
            </a>
          </div>
          <div>
            <span className="text-muted-foreground">创建交易 </span>
            <a
              href={explorerTx(cid, collection.created_tx_hash)}
              target="_blank"
              rel="noreferrer"
              className="font-mono text-primary underline-offset-4 hover:underline"
            >
              {shortHash(collection.created_tx_hash)}
            </a>
            <span className="ml-2 text-muted-foreground">block {collection.created_block_number}</span>
          </div>
        </CardContent>
      </Card>

      <Card className="border-white/10 bg-card/40">
        <CardHeader className="flex flex-row flex-wrap items-end justify-between gap-2">
          <div>
            <CardTitle className="text-base">Token 列表（库）</CardTitle>
            <CardDescription>
              共 {tokenTotal} 个 · 第 {tokenPage} 页，每页 {tokenPageSize}
            </CardDescription>
          </div>
          <div className="flex gap-2">
            <Button
              variant="outline"
              size="sm"
              disabled={tokenPage <= 1}
              onClick={() => setTokenPage((p) => Math.max(1, p - 1))}
            >
              上一页
            </Button>
            <Button
              variant="outline"
              size="sm"
              disabled={tokenPage * tokenPageSize >= tokenTotal}
              onClick={() => setTokenPage((p) => p + 1)}
            >
              下一页
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          {tokens.length === 0 ? (
            <p className="text-sm text-muted-foreground">暂无 Token 记录。</p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>token_id</TableHead>
                  <TableHead>owner</TableHead>
                  <TableHead>mint</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {tokens.map((t) => (
                  <TableRow key={t.id}>
                    <TableCell className="font-mono text-xs">{t.token_id}</TableCell>
                    <TableCell className="font-mono text-xs">
                      <a
                        href={explorerAddress(cid, t.owner_address)}
                        target="_blank"
                        rel="noreferrer"
                        className="text-primary underline-offset-4 hover:underline"
                      >
                        {shortHash(t.owner_address, 6, 4)}
                      </a>
                    </TableCell>
                    <TableCell className="font-mono text-xs">
                      {t.mint_tx_hash ? (
                        <a
                          href={explorerTx(cid, t.mint_tx_hash)}
                          target="_blank"
                          rel="noreferrer"
                          className="text-primary underline-offset-4 hover:underline"
                        >
                          {shortHash(t.mint_tx_hash)}
                        </a>
                      ) : (
                        "—"
                      )}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

    </div>
  );
}
