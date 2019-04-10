ALTER TABLE `chatusers` ADD `sex` INT NOT NULL DEFAULT 0;
ALTER TABLE `chatusers` ADD INDEX `sex_index` (`sex`);

ALTER TABLE `chatusers` ADD `country` VARCHAR(16);
ALTER TABLE `chatusers` ADD INDEX `country_index` (`country`);

ALTER TABLE `chatusers` ADD `province` VARCHAR(64);
ALTER TABLE `chatusers` ADD INDEX `province_index` (`province`);

ALTER TABLE `chatusers` ADD `city` VARCHAR(64);
ALTER TABLE `chatusers` ADD INDEX `city_index` (`city`);

ALTER TABLE `chatusers` ADD `signature` VARCHAR(64);
ALTER TABLE `chatusers` ADD INDEX `signature_index` (`signature`);

ALTER TABLE `chatusers` ADD `remark` VARCHAR(32);
ALTER TABLE `chatusers` ADD INDEX `remark_index` (`remark`);

ALTER TABLE `chatusers` ADD `label` VARCHAR(128);
ALTER TABLE `chatusers` ADD INDEX `label_index` (`label`);




