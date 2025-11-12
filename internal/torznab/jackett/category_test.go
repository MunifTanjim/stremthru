package jackett

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCategoryParent(t *testing.T) {
	for _, test := range []struct {
		cat, parent Category
	}{
		{CategoryTV_Anime, CategoryTV},
		{CategoryTV_HD, CategoryTV},
		{CategoryPC_MobileAndroid, CategoryPC},
		{CategoryOther_Hashed, CategoryOther},
	} {
		c := ParentCategory(test.cat)
		assert.Equal(t, test.parent, c)
	}
}
