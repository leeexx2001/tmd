package twitter

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/go-resty/resty/v2"
	"github.com/tidwall/gjson"
)

const (
	timelineTweet = iota
	timelineUser
)

func getInstructions(resp []byte, path string) (gjson.Result, error) {
	inst := gjson.GetBytes(resp, path)
	if !inst.Exists() {
		// 简化错误信息，避免输出原始响应数据
		return gjson.Result{}, fmt.Errorf("unable to get timeline data: the resource may not exist or be private")
	}
	return inst, nil
}

func getEntries(instructions gjson.Result) gjson.Result {
	for _, inst := range instructions.Array() {
		if inst.Get("type").String() == "TimelineAddEntries" {
			return inst.Get("entries")
		}
	}
	return gjson.Result{}
}

func getModuleItems(instructions gjson.Result) gjson.Result {
	for _, inst := range instructions.Array() {
		if inst.Get("type").String() == "TimelineAddToModule" {
			return inst.Get("moduleItems")
		}
	}
	return gjson.Result{}
}

func getNextCursorSafe(entries gjson.Result) (string, error) {
	if !entries.Exists() || !entries.IsArray() {
		return "", nil
	}
	array := entries.Array()
	if len(array) == 0 {
		return "", nil
	}

	for i := len(array) - 1; i >= 0; i-- {
		if array[i].Get("content.entryType").String() == "TimelineTimelineCursor" &&
			array[i].Get("content.cursorType").String() == "Bottom" {
			return array[i].Get("content.value").String(), nil
		}
	}

	return "", nil
}

func getItemContentFromModuleItem(moduleItem gjson.Result) (gjson.Result, error) {
	res := moduleItem.Get("item.itemContent")
	if !res.Exists() {
		return gjson.Result{}, fmt.Errorf("invalid ModuleItem: %s", moduleItem.String())
	}
	return res, nil
}

func getItemContentsFromEntry(entry gjson.Result) ([]gjson.Result, error) {
	content := entry.Get("content")
	switch content.Get("entryType").String() {
	case "TimelineTimelineModule":
		return content.Get("items.#.item.itemContent").Array(), nil
	case "TimelineTimelineItem":
		return []gjson.Result{content.Get("itemContent")}, nil
	default:
		return nil, fmt.Errorf("invalid entry: %s", entry.String())
	}
}

func getResults(itemContent gjson.Result, itemType int) (gjson.Result, error) {
	switch itemType {
	case timelineTweet:
		return itemContent.Get("tweet_results"), nil
	case timelineUser:
		return itemContent.Get("user_results"), nil
	default:
		return gjson.Result{}, fmt.Errorf("invalid itemContent: %s", itemContent.String())
	}
}

func getTimelineResp(ctx context.Context, api timelineApi, client *resty.Client) ([]byte, error) {
	url := makeUrl(api)
	log.Infoln("[getTimelineResp] making request to:", url)
	resp, err := client.R().SetContext(ctx).Get(url)
	if err != nil {
		log.Errorln("[getTimelineResp] request failed:", err)
		return nil, err
	}
	log.Infoln("[getTimelineResp] got response, status:", resp.Status())
	if err := CheckApiResp(resp.Body()); err != nil {
		log.Errorln("[getTimelineResp] API error:", err)
		return nil, err
	}
	log.Infoln("[getTimelineResp] response OK")
	return resp.Body(), nil
}

// 获取时间线 API 并返回所有 itemContent 和 底部 cursor
func getTimelineItemContents(ctx context.Context, api timelineApi, client *resty.Client, instPath string) ([]gjson.Result, string, error) {
	log.Infoln("[getTimelineItemContents] fetching timeline response")
	resp, err := getTimelineResp(ctx, api, client)
	if err != nil {
		log.Errorln("[getTimelineItemContents] failed to get timeline response:", err)
		return nil, "", err
	}
	log.Infoln("[getTimelineItemContents] got response, length:", len(resp))

	// is temporarily unavailable because it violates the Twitter Media Policy.
	// Protected User's following: Permission denied
	if string(resp) == "{\"data\":{\"user\":{}}}" {
		return nil, "", nil
	}
	instructions, err := getInstructions(resp, instPath)
	if err != nil {
		return nil, "", err
	}
	entries := getEntries(instructions)
	moduleItems := getModuleItems(instructions)
	if !entries.Exists() && !moduleItems.Exists() {
		return nil, "", nil
	}

	itemContents := make([]gjson.Result, 0)
	if entries.IsArray() {
		for _, entry := range entries.Array() {
			if entry.Get("content.entryType").String() != "TimelineTimelineCursor" {
				contents, err := getItemContentsFromEntry(entry)
				if err != nil {
					log.Debugln("getItemContentsFromEntry failed:", err)
					continue
				}
				itemContents = append(itemContents, contents...)
			}
		}
	}
	if moduleItems.IsArray() {
		for _, moduleItem := range moduleItems.Array() {
			content, err := getItemContentFromModuleItem(moduleItem)
			if err != nil {
				log.Debugln("getItemContentFromModuleItem failed:", err)
				continue
			}
			itemContents = append(itemContents, content)
		}
	}
	cursor, err := getNextCursorSafe(entries)
	if err != nil {
		return nil, "", err
	}
	return itemContents, cursor, nil
}

func getTimelineItemContentsTillEnd(ctx context.Context, api timelineApi, client *resty.Client, instPath string) ([]gjson.Result, error) {
	log.Infoln("[getTimelineItemContentsTillEnd] start fetching, instPath:", instPath)
	res := make([]gjson.Result, 0)
	pageCount := 0

	for {
		pageCount++
		log.Infoln("[getTimelineItemContentsTillEnd] fetching page", pageCount)
		page, next, err := getTimelineItemContents(ctx, api, client, instPath)
		if err != nil {
			log.Errorln("[getTimelineItemContentsTillEnd] failed on page", pageCount, ":", err)
			return nil, err
		}
		log.Infoln("[getTimelineItemContentsTillEnd] page", pageCount, "got", len(page), "items, next cursor:", next)

		if len(page) == 0 {
			log.Infoln("[getTimelineItemContentsTillEnd] empty page, stopping")
			break // empty page
		}

		res = append(res, page...)
		if next == "" {
			log.Infoln("[getTimelineItemContentsTillEnd] no more pages")
			break
		}
		api.SetCursor(next)
	}

	log.Infoln("[getTimelineItemContentsTillEnd] finished, total pages:", pageCount, "total items:", len(res))
	return res, nil
}
