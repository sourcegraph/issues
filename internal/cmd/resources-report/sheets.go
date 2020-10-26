package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

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

func updateSheet(ctx context.Context, sheetID string, resources Resources, highlighted int) (string, error) {
	client, err := sheets.NewService(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to init client: %w", err)
	}

	reportTime := time.Now()
	newPageID := reportTime.Unix()
	newPage := fmt.Sprintf("Report-%s", reportTime.UTC().Format(time.RFC3339))

	_, err = client.Spreadsheets.BatchUpdate(sheetID, &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			// generate new sheet for this report
			{AddSheet: &sheets.AddSheetRequest{
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
			{RepeatCell: &sheets.RepeatCellRequest{
				Range: &sheets.GridRange{
					SheetId:        newPageID,
					StartRowIndex:  1,
					EndRowIndex:    int64(highlighted) + 1,
					EndColumnIndex: int64(len(reportSheetHeaders)),
				},
				Cell: &sheets.CellData{
					UserEnteredFormat: &sheets.CellFormat{
						TextFormat: &sheets.TextFormat{Bold: true},
					},
				},
				Fields: "userEnteredFormat.textFormat.bold",
			}},
		},
	}).Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("failed to format report: %w", err)
	}

	// append report to new sheet
	_, err = client.Spreadsheets.Values.Update(sheetID, fmt.Sprintf("'%s'!A:Z", newPage), &sheets.ValueRange{
		Values: toSheetValues(resources),
	}).ValueInputOption("RAW").Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("failed to update report: %w", err)
	}

	return strconv.Itoa(int(newPageID)), nil
}
