package repo

import "go.yorun.ai/vine/internal/daemon/hub/src/server/repo/db/model"

const (
	metadataSeededName  = "is_seeded"
	metadataSeededValue = "true"
)

type MetadataRepo struct {
	Dao *model.MetadataDao `inject:""`
}

func (r *MetadataRepo) IsSeeded() bool {
	row, ok := r.Dao.ByName(metadataSeededName)
	return ok && row.Value == metadataSeededValue
}

func (r *MetadataRepo) MarkSeeded() {
	r.Dao.SaveByName(metadataSeededName, metadataSeededValue)
}
