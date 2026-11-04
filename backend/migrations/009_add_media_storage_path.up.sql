-- 投稿メディアは非公開オブジェクトとして保存し、表示のたびに署名付きURLを発行する。
-- 署名付きURLは期限切れになるためDBには保存せず、Storage 上のオブジェクトパスだけを持つ。
-- url は移行前に公開URLで保存した既存行のためだけに残す（新規行は空文字）。
ALTER TABLE media ADD COLUMN storage_path VARCHAR(500) NOT NULL DEFAULT '';
ALTER TABLE media ALTER COLUMN url SET DEFAULT '';
