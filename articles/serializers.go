package articles

type ArticleSerializer struct {
	Article ArticleOutput
}

type ArticleResponse struct {
	Title          string         `json:"title"`
	Description    string         `json:"description"`
	Body           string         `json:"body"`
	Slug           string         `json:"slug"`
	TagList        []string       `json:"tagList"`
	Favorited      bool           `json:"favorited"`
	FavoritesCount int64          `json:"favoritesCount"`
	Author         AuthorResponse `json:"author"`
	CreateAt       string         `json:"createdAt"`
	UpdatedAt      string         `json:"updatedAt"`
}

type AuthorResponse struct {
	Username  string `json:"username"`
	Bio       string `json:"bio"`
	Image     string `json:"image"`
	Following bool   `json:"following"`
}

func tagsToStrings(tags []TagModel) []string {
	result := make([]string, len(tags))
	for i, tag := range tags {
		result[i] = tag.Tag
	}
	return result
}

func imageToString(image *string) string {
	if image == nil {
		return ""
	}
	return *image
}

func (s ArticleSerializer) Response() ArticleResponse {
	return ArticleResponse{
		Title:          s.Article.Title,
		Description:    s.Article.Description,
		Body:           s.Article.Body,
		Slug:           s.Article.Slug,
		TagList:        tagsToStrings(s.Article.Tags),
		Favorited:      s.Article.Favorited,
		FavoritesCount: s.Article.FavoritesCount,
		Author: AuthorResponse{
			Username:  s.Article.Author.Username,
			Bio:       s.Article.Author.Bio,
			Image:     imageToString(s.Article.Author.Image),
			Following: s.Article.AuthorFollowing,
		},
		CreateAt:  s.Article.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt: s.Article.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

type CommentSerializer struct {
	Comment CommentOutput
}

type CommentResponse struct {
	ID        uint           `json:"id"`
	Body      string         `json:"body"`
	Author    AuthorResponse `json:"author"`
	CreatedAt string         `json:"createdAt"`
	UpdatedAt string         `json:"updatedAt"`
}

func (s CommentSerializer) Response() CommentResponse {
	return CommentResponse{
		ID:   s.Comment.ID,
		Body: s.Comment.Body,
		Author: AuthorResponse{
			Username:  s.Comment.Author.Username,
			Bio:       s.Comment.Author.Bio,
			Image:     imageToString(s.Comment.Author.Image),
			Following: false, // TODO: set following status
		},
		CreatedAt: s.Comment.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt: s.Comment.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}
