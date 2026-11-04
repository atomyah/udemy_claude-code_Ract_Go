-- storage_path だけを持つ行（url が空）は参照先を失うため、列と一緒に削除する。
-- Storage 上のオブジェクトは残るので、必要なら別途削除する。
DELETE FROM media WHERE url = '';
ALTER TABLE media ALTER COLUMN url DROP DEFAULT;
ALTER TABLE media DROP COLUMN storage_path;
