package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/sheets/v4"
)

var reportSheetHeaders = []interface{}{"Platform", "Type", "ID", "Location", "Owner", "Created", "Meta"}

func toSheetValues(resources Resources) [][]interface{} {
	values := make([][]interface{}, len(resources)+1)
	values[0] = reportSheetHeaders
	for i, resource := range resources {
		var meta string
		metaBytes, err := json.Marshal(resource.Meta)
		if err == nil {
			meta = string(metaBytes)
		}
		values[i+1] = []interface{}{
			resource.Platform,
			resource.Type,
			resource.Identifier,
			resource.Location,
			resource.Owner,
			resource.Created.UTC(),
			meta,
		}
	}
	return values
}

const generatedSheetPagePrefix = "Report-"

type updateSheetOptions struct {
	// First n rows to highlight
	HighlightedRows int
	// Prune sheets created more than the given duration ago
	PruneOlderThan time.Duration
	// Verbose log output toggle
	Verbose bool
}

func updateSheet(ctx context.Context, sheetID string, resources Resources, opts updateSheetOptions) (string, error) {
	client, err := sheets.NewService(ctx)
	if err != nil {
		return "", errors.Errorf("failed to init client: %w", err)
	}

	reportTime := time.Now()
	var sheetOps []*sheets.Request
	newPageID := reportTime.Unix()
	newPage := generatedSheetPagePrefix + reportTime.UTC().Format(time.RFC3339)

	// query for pages (sheets) we might need to delete
	if opts.PruneOlderThan > 0 {
		oldestSheet := reportTime.Add(-opts.PruneOlderThan)
		if opts.Verbose {
			log.Printf("pruning pages older than %s\n", oldestSheet)
		}
		mainSheet, err := client.Spreadsheets.Get(sheetID).Fields(googleapi.Field("sheets")).Context(ctx).Do()
		if err != nil {
			return "", errors.Errorf("unable to get sheet %q: %w", sheetID, err)
		}
		for _, page := range mainSheet.Sheets {
			if page == nil || page.Properties == nil {
				continue
			}
			if strings.HasPrefix(page.Properties.Title, generatedSheetPagePrefix) {
				// parse date out of the sheet id, which we set to be unix timestamps (see above)
				reportID := page.Properties.SheetId
				reportCreatedDate := time.Unix(reportID, 0)
				if reportCreatedDate.Before(oldestSheet) {
					sheetOps = append(sheetOps, &sheets.Request{
						DeleteSheet: &sheets.DeleteSheetRequest{
							SheetId: reportID,
						},
					})
					if opts.Verbose {
						log.Printf("adding op to delete sheet '%d' (%s older than delete threshold)\n",
							reportID, oldestSheet.Sub(reportCreatedDate))
					}
				}
			}
		}
	}

	// set up a new page (sheet) within the sheet if there are resources to report
	if len(resources) > 0 {
		if opts.Verbose {
			log.Printf("creating page %q (ID: %d) with %d rows higlighted\n", newPage, newPageID, opts.HighlightedRows)
		}
		sheetOps = append(sheetOps,
			// generate new sheet for this report
			&sheets.Request{AddSheet: &sheets.AddSheetRequest{
				Properties: &sheets.SheetProperties{
					SheetId: newPageID,
					Title:   newPage,
					Index:   1,
					// fix header: https://developers.google.com/sheets/api/samples/formatting#format_a_header_row
					GridProperties: &sheets.GridProperties{
						FrozenRowCount:          1,
						ColumnGroupControlAfter: true,
					},
				},
			}},
			// highlight cells for most recent entries
			&sheets.Request{RepeatCell: &sheets.RepeatCellRequest{
				Range: &sheets.GridRange{
					SheetId:        newPageID,
					StartRowIndex:  1,
					EndRowIndex:    int64(opts.HighlightedRows) + 1,
					EndColumnIndex: int64(len(reportSheetHeaders)),
				},
				Cell: &sheets.CellData{
					UserEnteredFormat: &sheets.CellFormat{
						TextFormat: &sheets.TextFormat{Bold: true},
					},
				},
				Fields: "userEnteredFormat.textFormat.bold",
			}})
	}

	// make the changes (if there are any)
	if len(sheetOps) > 0 {
		_, err = client.Spreadsheets.BatchUpdate(sheetID, &sheets.BatchUpdateSpreadsheetRequest{
			Requests: sheetOps,
		}).Context(ctx).Do()
		if err != nil {
			return "", errors.Errorf("failed to format report: %w", err)
		}
	} else if opts.Verbose {
		log.Println("no changes to make to sheet")
	}

	// append report to new sheet (if there are resources)
	if len(resources) > 0 {
		sheetValues := toSheetValues(resources)
		_, err = client.Spreadsheets.Values.Update(sheetID, fmt.Sprintf("'%s'!A:Z", newPage), &sheets.ValueRange{
			Values: sheetValues,
		}).ValueInputOption("RAW").Context(ctx).Do()
		if err != nil {
			return "", errors.Errorf("failed to update report values: %w", err)
		}
		if opts.Verbose {
			log.Printf("wrote %d resources to sheet %q", len(sheetValues), newPageID)
		}
	}

	return strconv.Itoa(int(newPageID)), nil
}
