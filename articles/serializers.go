package articles

type ArticleSerializer struct {
	Article ArticleModel
}

type ArticleResponse struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Body        string `json:"body"`
	Slug        string `json:"slug"`
}

func (serializer ArticleSerializer) Response() ArticleResponse {
	return ArticleResponse{
		Title:       serializer.Article.Title,
		Description: serializer.Article.Description,
		Body:        serializer.Article.Body,
		Slug:        serializer.Article.Slug,
	}
}
