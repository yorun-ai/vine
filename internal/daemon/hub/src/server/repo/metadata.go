package repo

import "go.yorun.ai/vine/internal/daemon/hub/src/server/repo/db/model"

const (
	metadataSeededName  = "is_seeded"
	metadataSeededValue = "true"
)

type DBMetadataRepo struct {
	Dao *model.MetadataDao `inject:""`
}

func (r *DBMetadataRepo) IsSeeded() bool {
	row, ok := r.Dao.ByName(metadataSeededName)
	return ok && row.Value == metadataSeededValue
}

func (r *DBMetadataRepo) MarkSeeded() {
	r.Dao.SaveByName(metadataSeededName, metadataSeededValue)
}
