#!/usr/bin/env python3
"""
Prepare GDPR text for parsing.

This script cleans and normalizes the GDPR text downloaded from EUR-Lex,
making it suitable for automated parsing.
"""

import re
import sys
import json


def clean_gdpr_text(input_path: str, output_path: str) -> dict:
    """
    Clean and normalize GDPR text.

    Returns statistics about the processed document.
    """
    with open(input_path, 'r', encoding='utf-8') as f:
        text = f.read()

    # Remove table formatting (ASCII box characters)
    text = re.sub(r'\+[-+]+\+', '', text)
    text = re.sub(r'\|', '', text)

    # Clean up excessive whitespace on each line
    lines = text.split('\n')
    lines = [re.sub(r'[ \t]+', ' ', line).strip() for line in lines]
    text = '\n'.join(lines)

    # Fix recital numbering - ensure proper format
    text = re.sub(r'^\((\d+)\)\s*', r'(\1) ', text, flags=re.MULTILINE)

    # Normalize chapter headings: "CHAPTER X" followed by title
    # Keep the blank line between them but ensure format is consistent
    text = re.sub(r'\n{3,}', '\n\n', text)

    # Remove empty lines at start
    text = text.lstrip('\n')

    # Write output
    with open(output_path, 'w', encoding='utf-8') as f:
        f.write(text)

    # Calculate statistics - only count actual article headings
    # Article headings in GDPR are "Article N" followed by title on next line
    article_pattern = r'^Article (\d+)$'
    article_matches = re.findall(article_pattern, text, re.MULTILINE)
    unique_articles = sorted(set(int(a) for a in article_matches))

    # Count chapters - "CHAPTER X" on its own line
    chapter_pattern = r'^CHAPTER ([IVX]+)$'
    chapter_matches = re.findall(chapter_pattern, text, re.MULTILINE)

    # Count recitals (numbered paragraphs in preamble)
    recital_pattern = r'^\((\d+)\) '
    recital_matches = re.findall(recital_pattern, text, re.MULTILINE)
    unique_recitals = sorted(set(int(r) for r in recital_matches))

    stats = {
        'total_lines': len(text.split('\n')),
        'total_chars': len(text),
        'chapters': len(chapter_matches),
        'chapter_list': chapter_matches,
        'articles': len(unique_articles),
        'article_range': f"{min(unique_articles)}-{max(unique_articles)}" if unique_articles else "none",
        'recitals': len(unique_recitals),
        'recital_range': f"{min(unique_recitals)}-{max(unique_recitals)}" if unique_recitals else "none",
    }

    return stats


def generate_expected_json(text_path: str, json_path: str) -> None:
    """
    Generate expected parse output JSON for validation.
    """
    with open(text_path, 'r', encoding='utf-8') as f:
        text = f.read()

    # Extract chapters with their titles
    # Format: "CHAPTER X\n\nTitle"
    chapter_pattern = r'^CHAPTER ([IVX]+)\n\n(.+?)(?=\n\n|$)'
    chapters = []
    for match in re.finditer(chapter_pattern, text, re.MULTILINE):
        chapters.append({
            'number': match.group(1),
            'title': match.group(2).strip()
        })

    # Extract articles with their titles
    # Format: "Article N\n\nTitle" or "Article N\nTitle"
    article_pattern = r'^Article (\d+)\n+([^\n]+)'
    articles = []
    for match in re.finditer(article_pattern, text, re.MULTILINE):
        article_num = int(match.group(1))
        title = match.group(2).strip()
        articles.append({
            'number': article_num,
            'title': title
        })

    # Remove duplicates and sort by number
    seen_articles = {}
    for art in articles:
        if art['number'] not in seen_articles:
            seen_articles[art['number']] = art
    articles = sorted(seen_articles.values(), key=lambda x: x['number'])

    # Extract definitions from Article 4
    definitions = []
    # Find Article 4 section - definitions are between Article 4 and Article 5
    art4_start = text.find('\nArticle 4\n')
    art5_start = text.find('\nArticle 5\n')
    if art4_start != -1 and art5_start != -1:
        art4_content = text[art4_start:art5_start]
        # Definitions are formatted like: (1) 'personal data' means...
        # Note: various quote styles used
        def_pattern = r"\((\d+)\)\s*['\u2018\u2019'\"']([^'\u2018\u2019'\"']+)['\u2018\u2019'\"']"
        for match in re.finditer(def_pattern, art4_content):
            definitions.append({
                'number': int(match.group(1)),
                'term': match.group(2).strip()
            })

    expected = {
        'regulation': 'GDPR',
        'full_name': 'Regulation (EU) 2016/679',
        'statistics': {
            'chapters': len(chapters),
            'articles': len(articles),
            'definitions': len(definitions)
        },
        'chapters': chapters,
        'articles': articles,
        'definitions': definitions
    }

    with open(json_path, 'w', encoding='utf-8') as f:
        json.dump(expected, f, indent=2, ensure_ascii=False)


def main():
    if len(sys.argv) < 3:
        print(f"Usage: {sys.argv[0]} <input_file> <output_file> [--generate-expected <json_file>]")
        sys.exit(1)

    input_path = sys.argv[1]
    output_path = sys.argv[2]

    print(f"Processing {input_path}...")
    stats = clean_gdpr_text(input_path, output_path)

    print(f"\nGDPR Text Statistics:")
    print(f"  Total lines: {stats['total_lines']}")
    print(f"  Total characters: {stats['total_chars']}")
    print(f"  Chapters: {stats['chapters']} ({', '.join(stats['chapter_list'])})")
    print(f"  Articles: {stats['articles']} (range: {stats['article_range']})")
    print(f"  Recitals: {stats['recitals']} (range: {stats['recital_range']})")
    print(f"\nOutput written to {output_path}")

    # Generate expected JSON if requested
    if len(sys.argv) >= 5 and sys.argv[3] == '--generate-expected':
        json_path = sys.argv[4]
        print(f"\nGenerating expected output JSON...")
        generate_expected_json(output_path, json_path)
        print(f"Expected output written to {json_path}")


if __name__ == '__main__':
    main()
