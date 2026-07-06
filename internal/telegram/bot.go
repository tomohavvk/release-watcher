package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/shadowy-pycoder/release-watcher/internal/domain"
	gh "github.com/shadowy-pycoder/release-watcher/internal/github"
	"github.com/shadowy-pycoder/release-watcher/internal/poller"
	"github.com/shadowy-pycoder/release-watcher/internal/repository"
)

type Bot struct {
	token  string
	apiURL string
	client *http.Client
	repos  *repository.All
	github *gh.Client
	poller *poller.Poller
}

func New(token string, repos *repository.All, github *gh.Client, poller *poller.Poller) *Bot {
	return &Bot{
		token:  token,
		apiURL: "https://api.telegram.org/bot" + token,
		client: &http.Client{Timeout: 60 * time.Second},
		repos:  repos,
		github: github,
		poller: poller,
	}
}

func (b *Bot) Start(ctx context.Context) {
	b.setCommands(ctx)
	slog.Info("telegram bot started")

	offset := int64(0)
	for {
		if ctx.Err() != nil {
			slog.Info("telegram bot stopped")
			return
		}

		updates, err := b.getUpdates(ctx, offset, 30)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("telegram: failed to get updates", "error", err)
			time.Sleep(5 * time.Second)
			continue
		}

		for _, u := range updates {
			b.handleUpdate(ctx, u)
			offset = u.UpdateID + 1
		}
	}
}

func (b *Bot) SendCode(ctx context.Context, chatID int64, code string) error {
	text := fmt.Sprintf("🔐 Код для входу: <code>%s</code>\n\nДійсний 5 хвилин.", code)
	return b.sendMessage(ctx, chatID, text, "HTML")
}

func (b *Bot) NotifyNewRelease(ctx context.Context, rel *domain.Release) {
	chatIDs, err := b.repos.User.GetTelegramRecipientsForRepo(ctx, rel.RepoID, rel.PublishedAt)
	if err != nil {
		slog.Error("telegram: failed to get recipients", "error", err)
		return
	}

	if len(chatIDs) == 0 {
		return
	}

	text := formatRelease(rel)
	for _, chatID := range chatIDs {
		if err := b.sendMessage(ctx, chatID, text, "HTML"); err != nil {
			slog.Error("telegram: failed to send notification", "chat_id", chatID, "error", err)
		}
	}
}

func (b *Bot) handleUpdate(ctx context.Context, u update) {
	if u.Message == nil || u.Message.Text == "" {
		return
	}

	chatID := u.Message.Chat.ID
	text := strings.TrimSpace(u.Message.Text)
	userName := u.Message.From.Username
	if userName == "" {
		userName = u.Message.From.FirstName
	}

	switch {
	case text == "/start" || text == "/help":
		b.handleStart(ctx, chatID)
	case text == "/list":
		b.handleList(ctx, chatID, userName)
	case strings.HasPrefix(text, "/remove"):
		b.handleRemove(ctx, chatID, userName, strings.TrimSpace(strings.TrimPrefix(text, "/remove")))
	case strings.HasPrefix(text, "/mute"):
		b.handleMute(ctx, chatID, userName, strings.TrimSpace(strings.TrimPrefix(text, "/mute")))
	case strings.HasPrefix(text, "/unmute"):
		b.handleUnmute(ctx, chatID, userName, strings.TrimSpace(strings.TrimPrefix(text, "/unmute")))
	case isGitHubURL(text):
		b.handleSubscribe(ctx, chatID, userName, text)
	default:
		b.sendMessage(ctx, chatID, "Надішли посилання на GitHub (github.com/owner або github.com/owner/repo).\nНатисни /help для довідки.", "")
	}
}

func (b *Bot) handleStart(ctx context.Context, chatID int64) {
	text := `<b>Release Watcher Bot</b>

Слідкую за новими релізами та тегами на GitHub і повідомляю тебе.

<b>Як користуватись:</b>
Надішли посилання на GitHub щоб підписатись:

<code>https://github.com/singnet</code> — всі репо організації
<code>https://github.com/singnet/snet-daemon</code> — конкретний репо

<b>Команди:</b>
/list — твої підписки
/remove &lt;url&gt; — відписатись
/mute &lt;url&gt; — замутити репо
/unmute &lt;url&gt; — розмутити
/help — ця довідка`

	b.sendMessage(ctx, chatID, text, "HTML")
}

func (b *Bot) handleList(ctx context.Context, chatID int64, userName string) {
	user, err := b.repos.User.GetOrCreateByTelegram(ctx, chatID, userName)
	if err != nil {
		b.sendMessage(ctx, chatID, "Помилка. Спробуй ще раз.", "")
		return
	}

	orgs, _ := b.repos.Organization.ListByUser(ctx, user.ID)
	repos, _ := b.repos.Repository.ListFollowedByUser(ctx, user.ID)
	mutes, _ := b.repos.Mute.ListByUser(ctx, user.ID)

	if len(orgs) == 0 && len(repos) == 0 {
		b.sendMessage(ctx, chatID, "У тебе поки немає підписок.\nНадішли посилання на GitHub щоб підписатись.", "")
		return
	}

	var sb strings.Builder
	if len(orgs) > 0 {
		sb.WriteString("<b>Організації:</b>\n")
		for _, o := range orgs {
			sb.WriteString(fmt.Sprintf("• <a href=\"https://github.com/%s\">%s</a>\n", o.Name, o.Name))
		}
	}

	if len(repos) > 0 {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("<b>Репозиторії:</b>\n")
		for _, r := range repos {
			sb.WriteString(fmt.Sprintf("• <a href=\"https://github.com/%s\">%s</a>\n", r.FullName, r.FullName))
		}
	}

	if len(mutes) > 0 {
		sb.WriteString("\n<b>Замучені:</b>\n")
		for _, m := range mutes {
			sb.WriteString(fmt.Sprintf("• %s\n", m.RepoName))
		}
	}

	b.sendMessage(ctx, chatID, sb.String(), "HTML")
}

func (b *Bot) handleSubscribe(ctx context.Context, chatID int64, userName, text string) {
	owner, repo, err := parseGitHubURL(text)
	if err != nil {
		b.sendMessage(ctx, chatID, "Невалідне посилання. Приклад: https://github.com/singnet", "")
		return
	}

	user, err := b.repos.User.GetOrCreateByTelegram(ctx, chatID, userName)
	if err != nil {
		b.sendMessage(ctx, chatID, "Помилка створення користувача.", "")
		return
	}

	if repo == "" {
		b.subscribeOrg(ctx, chatID, user, owner)
	} else {
		b.subscribeRepo(ctx, chatID, user, owner, repo)
	}
}

func (b *Bot) subscribeOrg(ctx context.Context, chatID int64, user *domain.User, owner string) {
	if !b.github.ValidateOwner(ctx, owner) {
		b.sendMessage(ctx, chatID, fmt.Sprintf("Організацію/користувача <b>%s</b> не знайдено на GitHub.", owner), "HTML")
		return
	}

	org, err := b.repos.Organization.FindOrCreate(ctx, owner)
	if err != nil {
		b.sendMessage(ctx, chatID, "Помилка збереження.", "")
		return
	}

	if err := b.repos.Organization.Follow(ctx, user.ID, org.ID); err != nil {
		b.sendMessage(ctx, chatID, "Помилка підписки.", "")
		return
	}

	b.poller.Trigger()
	b.sendMessage(ctx, chatID, fmt.Sprintf("Підписано на всі репо <b>%s</b>", owner), "HTML")
}

func (b *Bot) subscribeRepo(ctx context.Context, chatID int64, user *domain.User, owner, repoName string) {
	fullName := owner + "/" + repoName
	if !b.github.ValidateRepo(ctx, fullName) {
		b.sendMessage(ctx, chatID, fmt.Sprintf("Репозиторій <b>%s</b> не знайдено на GitHub.", fullName), "HTML")
		return
	}

	org, err := b.repos.Organization.FindOrCreate(ctx, owner)
	if err != nil {
		b.sendMessage(ctx, chatID, "Помилка збереження.", "")
		return
	}

	repo, err := b.repos.Repository.Upsert(ctx, org.ID, repoName, fullName)
	if err != nil {
		b.sendMessage(ctx, chatID, "Помилка збереження.", "")
		return
	}

	if err := b.repos.Repository.FollowRepo(ctx, user.ID, repo.ID); err != nil {
		b.sendMessage(ctx, chatID, "Помилка підписки.", "")
		return
	}

	b.poller.Trigger()
	b.sendMessage(ctx, chatID, fmt.Sprintf("Підписано на <b>%s</b>", fullName), "HTML")
}

func (b *Bot) handleRemove(ctx context.Context, chatID int64, userName, urlText string) {
	if urlText == "" {
		b.sendMessage(ctx, chatID, "Вкажи URL: /remove https://github.com/owner або /remove https://github.com/owner/repo", "")
		return
	}

	owner, repoName, err := parseGitHubURL(urlText)
	if err != nil {
		b.sendMessage(ctx, chatID, "Невалідне посилання.", "")
		return
	}

	user, err := b.repos.User.GetOrCreateByTelegram(ctx, chatID, userName)
	if err != nil {
		b.sendMessage(ctx, chatID, "Помилка.", "")
		return
	}

	if repoName == "" {
		org, err := b.repos.Organization.FindOrCreate(ctx, owner)
		if err != nil {
			b.sendMessage(ctx, chatID, "Організацію не знайдено.", "")
			return
		}
		b.repos.Organization.Unfollow(ctx, user.ID, org.ID)
		b.sendMessage(ctx, chatID, fmt.Sprintf("Відписано від <b>%s</b>", owner), "HTML")
	} else {
		fullName := owner + "/" + repoName
		repo, err := b.repos.Repository.GetByFullName(ctx, fullName)
		if err != nil {
			b.sendMessage(ctx, chatID, "Репозиторій не знайдено.", "")
			return
		}
		b.repos.Repository.UnfollowRepo(ctx, user.ID, repo.ID)
		b.sendMessage(ctx, chatID, fmt.Sprintf("Відписано від <b>%s</b>", fullName), "HTML")
	}
}

func (b *Bot) handleMute(ctx context.Context, chatID int64, userName, urlText string) {
	if urlText == "" {
		b.sendMessage(ctx, chatID, "Вкажи URL репо: /mute https://github.com/owner/repo", "")
		return
	}

	owner, repoName, err := parseGitHubURL(urlText)
	if err != nil || repoName == "" {
		b.sendMessage(ctx, chatID, "Вкажи повний URL репозиторію: /mute https://github.com/owner/repo", "")
		return
	}

	user, err := b.repos.User.GetOrCreateByTelegram(ctx, chatID, userName)
	if err != nil {
		b.sendMessage(ctx, chatID, "Помилка.", "")
		return
	}

	fullName := owner + "/" + repoName
	repo, err := b.repos.Repository.GetByFullName(ctx, fullName)
	if err != nil {
		b.sendMessage(ctx, chatID, "Репозиторій не знайдено в базі.", "")
		return
	}

	b.repos.Mute.Mute(ctx, user.ID, repo.ID)
	b.sendMessage(ctx, chatID, fmt.Sprintf("Замучено <b>%s</b>", fullName), "HTML")
}

func (b *Bot) handleUnmute(ctx context.Context, chatID int64, userName, urlText string) {
	if urlText == "" {
		b.sendMessage(ctx, chatID, "Вкажи URL репо: /unmute https://github.com/owner/repo", "")
		return
	}

	owner, repoName, err := parseGitHubURL(urlText)
	if err != nil || repoName == "" {
		b.sendMessage(ctx, chatID, "Вкажи повний URL репозиторію: /unmute https://github.com/owner/repo", "")
		return
	}

	user, err := b.repos.User.GetOrCreateByTelegram(ctx, chatID, userName)
	if err != nil {
		b.sendMessage(ctx, chatID, "Помилка.", "")
		return
	}

	fullName := owner + "/" + repoName
	repo, err := b.repos.Repository.GetByFullName(ctx, fullName)
	if err != nil {
		b.sendMessage(ctx, chatID, "Репозиторій не знайдено в базі.", "")
		return
	}

	b.repos.Mute.Unmute(ctx, user.ID, repo.ID)
	b.sendMessage(ctx, chatID, fmt.Sprintf("Розмучено <b>%s</b>", fullName), "HTML")
}

// Telegram API

type update struct {
	UpdateID int64    `json:"update_id"`
	Message  *message `json:"message"`
}

type message struct {
	From *tgUser `json:"from"`
	Chat *chat   `json:"chat"`
	Text string  `json:"text"`
}

type tgUser struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	Username  string `json:"username"`
}

type chat struct {
	ID int64 `json:"id"`
}

func (b *Bot) getUpdates(ctx context.Context, offset int64, timeout int) ([]update, error) {
	reqURL := fmt.Sprintf("%s/getUpdates?offset=%d&timeout=%d", b.apiURL, offset, timeout)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		OK     bool     `json:"ok"`
		Result []update `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Result, nil
}

func (b *Bot) sendMessage(ctx context.Context, chatID int64, text, parseMode string) error {
	params := url.Values{
		"chat_id":                  {fmt.Sprintf("%d", chatID)},
		"text":                     {text},
		"disable_web_page_preview": {"true"},
	}
	if parseMode != "" {
		params.Set("parse_mode", parseMode)
	}

	reqURL := b.apiURL + "/sendMessage?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return err
	}

	resp, err := b.client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (b *Bot) setCommands(ctx context.Context) {
	commands := []struct {
		Command     string `json:"command"`
		Description string `json:"description"`
	}{
		{"list", "Мої підписки"},
		{"remove", "Відписатись"},
		{"mute", "Замутити репо"},
		{"unmute", "Розмутити репо"},
		{"help", "Довідка"},
	}

	data, _ := json.Marshal(commands)
	params := url.Values{"commands": {string(data)}}
	reqURL := b.apiURL + "/setMyCommands?" + params.Encode()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	resp, err := b.client.Do(req)
	if err != nil {
		slog.Error("telegram: failed to set commands", "error", err)
		return
	}
	resp.Body.Close()
}

// Helpers

func parseGitHubURL(text string) (owner, repo string, err error) {
	text = strings.TrimSpace(text)
	text = strings.TrimSuffix(text, "/")

	if !strings.Contains(text, "github.com") {
		return "", "", fmt.Errorf("not a github url")
	}

	if !strings.HasPrefix(text, "http") {
		text = "https://" + text
	}

	u, err := url.Parse(text)
	if err != nil {
		return "", "", err
	}

	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		return "", "", fmt.Errorf("empty path")
	}

	if (parts[0] == "orgs" || parts[0] == "users") && len(parts) >= 2 {
		parts = parts[1:]
	}

	owner = parts[0]
	if len(parts) >= 2 && parts[1] != "" {
		repo = parts[1]
	}
	return owner, repo, nil
}

func isGitHubURL(text string) bool {
	return strings.Contains(text, "github.com/")
}

func formatRelease(rel *domain.Release) string {
	var emoji, typeLabel string
	if rel.Type == "release" {
		emoji = "\U0001f4e6"
		typeLabel = "Release"
	} else {
		emoji = "\U0001f3f7"
		typeLabel = "Tag"
	}

	name := rel.TagName
	if rel.Name != "" && rel.Name != rel.TagName {
		name = rel.TagName + " — " + rel.Name
	}

	return fmt.Sprintf(
		"%s <b>%s</b> [%s]\n\n<code>%s</code>\n\n<a href=\"%s\">View on GitHub</a>",
		emoji, rel.RepoName, typeLabel, name, rel.HTMLURL,
	)
}
