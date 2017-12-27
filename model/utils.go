package model

import (
	"fmt"

	"github.com/go-xorm/xorm"
	"github.com/satori/go.uuid"
)

var (
	db                 *xorm.Engine
	developerInsertSQL = "INSERT INTO `users` VALUES ('12b7f9d3-f8af-4690-b72c-8cb58bc6a6f6','masutech16','example@trap.jp','bea362d60712001c3b5892feb738a75f28155e401cf4c69199276507a4d6e9964ef8db6a31987aa64d2123a760eb62e396914559fa76de977b11514ed9590d44','274d017297a662d4b70af98213ce','empty',1,'2017-12-27 14:29:58.906939','2017-12-27 14:29:58.906941'),('2544aab2-6069-4e81-9e3f-127adefa1de5','clk','example@trap.jp','e45d19fa24320c4c5b0ce9d41665d1323f854aab96b4838957eda022f8ae15286abbc11f236a6d24ecc1995c7a7f622744211d34a539d3208ef1c3f1375829cc','8f3c412e29e1fca566bc57682c5a','empty',1,'2017-12-27 14:29:59.290904','2017-12-27 14:29:59.290907'),('340b3ade-eda4-4cba-8f56-5df3ead69a1c','sigma','example@trap.jp','0b4e827213dd0d811c17e3d913a4760839ac4fc722de52e5c252fa51d5cbcb0b9789f63ddb28ee6b18fe22ec60d8ac9189d9180f73039805865c5bef633012c1','57a1fa9acb791b703753a4d490bf','empty',1,'2017-12-27 14:29:59.536063','2017-12-27 14:29:59.536064'),('87be6381-1ba2-4df9-8fb8-fd5950dc9050','yasu','example@trap.jp','5c048cc96ea431245f37c16bfcc8425b798015ca92b7114d41331d93db8bfa2d4bfc8de1fb98aa9fc1f4110eb832d24a68f45ad8ad1cc819b5fbf8b47f7e0253','f1b1629e94cb875d190e3bf9aa36','empty',1,'2017-12-27 14:29:59.02753','2017-12-27 14:29:59.027532'),('89b1a4a5-9bc3-4b58-96b1-ff37769630d2','to-hutohu','example@trap.jp','fcdddc6eaf7e1ff0c00b0cadd61e9c70c796d4e889c0b8c967a62328792d9bc1abde7db096a2d1d0c48d947b20df474525e04b78a0b3e6fdac6ae9061d1fba93','4fc19ccba04e582415d34ac07559','empty',1,'2017-12-27 14:29:59.930179','2017-12-27 14:29:59.930181'),('9330418f-34d0-4c30-a4ef-3a0af09f8beb','kaz','example@trap.jp','678944f61ecf3bf364bd999a8894c1ba829453b2117e9732a4cf99122914e0ca5a27da06b955f2e7cec406322a9de9b859ed095d56367967eccc81620eca87cd','0ad307419dbb83025dc848609356','empty',1,'2017-12-27 14:29:58.662261','2017-12-27 14:29:58.662263'),('94b2f81c-08c3-4b3b-93ef-061222ca7e3b','ninja','example@trap.jp','a2413457b874952b16a2827b8d8b11c41e8b2cd5fb569f943ee5716609a882896ab15527ee03f80d056d31cad1ecb8a67f80fa4f5d58b0b2767dcc8c2f700821','5549707ca3b58b728b0eae00fe0f','empty',1,'2017-12-27 14:29:59.675515','2017-12-27 14:29:59.675518'),('95bfe69e-670b-421e-ade1-d9cb8dbbbd98','PS6S','example@trap.jp','ffe13c0157fdaff26f4a830ac0b228dd054eda2f14da3f246caf9c6ad098ab7847a3c21c42950012ad949bfe48968e92a5d2d12259ec351219654263daa76ae0','6752fb6faef4f2f88ef5351594b1','empty',1,'2017-12-27 14:29:58.792511','2017-12-27 14:29:58.792513'),('b3718725-1c1a-4e90-af31-846688a38aab','takashi_trap','example@trap.jp','6a0f402efe5ed42b57c4adbfd6e3614cb6cd189b6f600ad9f20878949a2eb4f58b35b975316626d2c475bbab0c7c54e56b029bb7676ee5b3cdbe1ccb3c5482bc','58e818c71471cc4ea9e00c5a6e4d','empty',1,'2017-12-27 14:29:59.796984','2017-12-27 14:29:59.796986'),('c50c8ea8-e8c6-499c-844c-7a32a2e787b8','Sahil','example@trap.jp','756eb7598376b64956eeefb33fe13ac0bc7fd3d5e57ad87ea0780af8ccd33adafe14cbd63d099f53395dc505fd7e85d6e4a002c83ea5a1fb878489d926721d32','f0b8fad0b322948dd18e9f8f71b6','empty',1,'2017-12-27 14:29:59.153088','2017-12-27 14:29:59.15309'),('dc5034fc-590b-4d2f-8847-c0f9a8e29255','JichouP','example@trap.jp','e36e64574de64ec4d2e0bebf6d012439cd0a840d99e4d4015f4cdff9e158f74159c684fec4d9924f93c2c5814a053de4caf54cdd33090ac7c85174ee02c350a6','3b30dfc571bda95ffac44cbdfce7','empty',1,'2017-12-27 14:29:59.421103','2017-12-27 14:29:59.421105');"
)

// SetXORMEngine DBにxormのエンジンを設定する
func SetXORMEngine(engine *xorm.Engine) {
	db = engine
}

// SyncSchema : テーブルと構造体を同期させる関数
// モデルを追加したら各自ここに追加しなければいけない
func SyncSchema() error {
	if err := db.Sync(new(Channel)); err != nil {
		return fmt.Errorf("Failed to sync Channels: %v", err)
	}

	if err := db.Sync(new(UsersPrivateChannel)); err != nil {
		return fmt.Errorf("Failed to sync UsersPrivateChannels: %v", err)
	}

	if err := db.Sync(&Message{}); err != nil {
		return fmt.Errorf("Failed to sync Messages Table: %v", err)
	}

	if err := db.Sync(&User{}); err != nil {
		return fmt.Errorf("Failed to sync Users Table: %v", err)
	}

	db.Exec(developerInsertSQL)
	return nil
}

// CreateUUID UUIDを生成する
func CreateUUID() string {
	return uuid.NewV4().String()
}
