ALTER TABLE `chatusers` DROP INDEX `sex_index`;
ALTER TABLE `chatusers` DROP COLUMN `sex`;

ALTER TABLE `chatusers` DROP INDEX `country_index`;
ALTER TABLE `chatusers` DROP COLUMN `country`;

ALTER TABLE `chatusers` DROP INDEX `province_index`;
ALTER TABLE `chatusers` DROP COLUMN `province`;

ALTER TABLE `chatusers` DROP INDEX `city_index`;
ALTER TABLE `chatusers` DROP COLUMN `city`;

ALTER TABLE `chatusers` DROP INDEX `signature_index`;
ALTER TABLE `chatusers` DROP COLUMN`signature`;

ALTER TABLE `chatusers` DROP INDEX `remark_index`;
ALTER TABLE `chatusers` DROP COLUMN `remark`;

ALTER TABLE `chatusers` DROP INDEX `label_index`;
ALTER TABLE `chatusers` DROP COLUMN `label`;

