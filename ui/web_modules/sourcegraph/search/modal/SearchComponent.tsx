import * as React from "react";

import {colors} from "sourcegraph/components/jsStyles/colors";
import {modal_result} from "sourcegraph/search/modal/SearchModalStyle.css";

const smallFont = 12.75;

const ViewMore = ({expandCategory, type}) => <a style={{
		textTransform: "uppercase",
		fontSize: smallFont,
		fontWeight: "bold",
		display: "block",
		textAlign: "center",
		color: colors.blue3(),
	}} onClick={expandCategory}>
	View more {type}
</a>;

const ResultRow = ({title, description, index, length}, categoryIndex, itemIndex, selected, delegate, scrollIntoView) => {
	return (
		<a key={itemIndex} className={modal_result}
			ref={node => { if (scrollIntoView && node && selected) {
			// Nonstandard, but improves experience in Chrome.
			(node as any).scrollIntoViewIfNeeded(false);
		}}}
		style={{
			borderRadius: 3,
			padding: "8px 16px",
			margin: "0 8px 8px 8px",
			backgroundColor: selected ? colors.blue2() : colors.coolGray1(.5),
			display: "block",
			color: colors.white(),
		}}
		onClick={() => delegate.select(categoryIndex, itemIndex)}>
		{length ? <div>
			<span>{title.substr(0, index)}</span>
			<span style={{fontWeight: "bold"}}>{title.substr(index, length)}</span>
			<span>{title.substr(index + length)}</span>
		</div> :
		<div style={{color: colors.white()}}>
			{title}
		</div>
		}
		<div style={{fontSize: smallFont, color: selected ? colors.white() : colors.coolGray3()}}>
			{description}
		</div>
		</a>
	);
};

const ResultCategory = ({title, results, isLoading, selected, delegate, categoryIndex, limit, expandCategory, scrollIntoView}) => {
	if (isLoading) {
		return (
			<div style={{padding: "14px 0"}}>
				<span style={{color: colors.coolGray3()}}>{title} (loading...)</span>
			</div>
		);
	}
	if (results.length === 0) {
		return <div></div>;
	}
	const total = results.length;
	results = results.slice(0, limit);
	return <div style={{padding: "14px 0"}}>
		<div style={{
			paddingBottom: "0.5em",
			color: colors.coolGray3(),
		}}>
			{title}
		</div>
		{results.map((result, index) =>
			ResultRow(result, categoryIndex, index, (index === selected), delegate, scrollIntoView)
		)}
		{total > limit ? <ViewMore expandCategory={expandCategory} type={title} /> : null}
	</div>;
};

export const ResultCategories = ({categories, selection, delegate, limits, scrollIntoView}) => {
	let loadingOrFound = false;
	categories.forEach(category => {
		if (category.IsLoading || (category.Results && category.Results.length)) {
			loadingOrFound = true;
		}
	});
	if (!loadingOrFound) {
		return <div style={{padding: "14px 0", color: colors.white(), textAlign: "center"}}>No results found</div>;
	}
	let sections: JSX.Element[] = [];
	categories.forEach((category, i) => {
		let selected = -1;
		if (i === selection.category) {
			selected = selection.row;
		}
		sections.push(<ResultCategory
						key={category.Title}
						limit={limits[i]}
						isLoading={category.IsLoading}
						categoryIndex={i}
						title={category.Title}
						selected={selected}
						results={category.Results}
						delegate={delegate}
						scrollIntoView={scrollIntoView}
						expandCategory={delegate.expand(i)} />);
	});
	return <div style={{overflow: "auto"}}>
		{sections}
	</div>;
};
