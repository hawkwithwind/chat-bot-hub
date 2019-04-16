ALTER TABLE `chatusers` DROP INDEX `lastsendat_index`;
ALTER TABLE `chatusers` DROP COLUMN `lastsendat`;

ALTER TABLE `chatgroups` DROP INDEX `lastsendat_index`;
ALTER TABLE `chatgroups` DROP COLUMN `lastsendat`;
