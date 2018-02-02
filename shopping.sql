/*
 Navicat Premium Data Transfer

 Source Server         : localhost
 Source Server Type    : MySQL
 Source Server Version : 50720
 Source Host           : localhost
 Source Database       : shopping

 Target Server Type    : MySQL
 Target Server Version : 50720
 File Encoding         : utf-8

 Date: 02/02/2018 15:31:05 PM
*/

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- ----------------------------
--  Table structure for `jd`
-- ----------------------------
DROP TABLE IF EXISTS `jd`;
CREATE TABLE `jd` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT '自增编号',
  `sku` bigint(20) unsigned NOT NULL COMMENT '商品编号',
  `price` double NOT NULL COMMENT '价格',
  `content` varchar(4096) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL DEFAULT '' COMMENT '内容',
  `jd_price` varchar(1024) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL DEFAULT '' COMMENT '京东价格',
  `jd_promotion` blob NOT NULL COMMENT '京东促销',
  `jd_page_config` blob NOT NULL COMMENT '京东页面配置',
  `record_timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '记录时间戳',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ----------------------------
--  Table structure for `sku`
-- ----------------------------
DROP TABLE IF EXISTS `sku`;
CREATE TABLE `sku` (
  `sku` bigint(20) unsigned NOT NULL COMMENT '商品编号',
  `priority` int(10) unsigned NOT NULL COMMENT '优先级',
  `min_price` double NOT NULL DEFAULT '0' COMMENT '最低价',
  `max_price` double NOT NULL DEFAULT '0' COMMENT '最高价',
  PRIMARY KEY (`sku`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

SET FOREIGN_KEY_CHECKS = 1;
