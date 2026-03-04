#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
[ -f "$SCRIPT_DIR/.env" ] && source "$SCRIPT_DIR/.env"
PORT="${HTTP_PORT:-8552}"
BASE="http://localhost:$PORT"

# Colors for output
GREEN='\033[0;32m'
CYAN='\033[0;36m'
NC='\033[0m'

header() {
    echo ""
    echo -e "${CYAN}=== $1 ===${NC}"
    echo ""
}

pretty() {
    python3 -m json.tool 2>/dev/null || cat
}

# --- Test cases ---

test_lookup_article() {
    local ref="$1"
    header "Lookup: $ref"
    curl -s "$BASE/api/article/$ref" | pretty
}

test_classify() {
    local name="$1"
    local desc="$2"
    header "Classify: $name"
    curl -s -X POST "$BASE/api/classify" \
        -H 'Content-Type: application/json' \
        -d "{\"description\":\"$desc\"}" | pretty
}

test_prohibited() {
    local name="$1"
    local desc="$2"
    header "Prohibited check: $name"
    curl -s -X POST "$BASE/api/prohibited" \
        -H 'Content-Type: application/json' \
        -d "{\"description\":\"$desc\"}" | pretty
}

test_checklist() {
    local name="$1"
    local desc="$2"
    header "Full checklist: $name"
    echo -e "${GREEN}Running 5-stage pipeline (may take 10-15 sec)...${NC}"
    local response
    response=$(curl -s -X POST "$BASE/api/checklist" \
        -H 'Content-Type: application/json' \
        -d "{\"description\":\"$desc\"}")
    # Print the markdown checklist directly
    echo "$response" | python3 -c "import sys,json; print(json.load(sys.stdin).get('checklist',''))" 2>/dev/null || echo "$response" | pretty
}

# --- Named test scenarios ---

case "${1:-}" in

    # Quick lookups (no LLM)
    article-5)
        test_lookup_article "article_5"
        ;;
    article-6)
        test_lookup_article "article_6"
        ;;
    article-3)
        test_lookup_article "article_3"
        ;;
    recital-47)
        test_lookup_article "recital_47"
        ;;
    annex-3)
        test_lookup_article "annex_3"
        ;;

    # Classification tests
    cv-screening)
        test_classify "CV Screening" \
            "An AI tool that screens CVs and ranks job candidates for recruitment"
        ;;
    social-scoring)
        test_classify "Social Scoring" \
            "A government system that scores citizens based on their social behavior and restricts access to services"
        ;;
    chatbot)
        test_classify "Customer Chatbot" \
            "A customer service chatbot for an e-commerce website"
        ;;
    spam-filter)
        test_classify "Spam Filter" \
            "An AI spam filter for email"
        ;;
    biometrics)
        test_classify "Biometric ID" \
            "A facial recognition system used at airport border control to verify traveler identity"
        ;;
    education)
        test_classify "Education AI" \
            "An AI system that grades student exams and determines university admissions"
        ;;
    law-enforcement)
        test_classify "Predictive Policing" \
            "A predictive policing system that identifies high-crime areas and suggests patrol routes"
        ;;
    foundation-model)
        test_classify "Foundation Model" \
            "A large language model trained on internet text, offered as an API for general-purpose use"
        ;;

    # Prohibited practices check
    emotion-workplace)
        test_prohibited "Emotion in Workplace" \
            "An AI system that monitors employee emotions via webcam to evaluate their productivity"
        ;;
    facial-scraping)
        test_prohibited "Facial Scraping" \
            "An AI that scrapes social media photos to build a facial recognition database for commercial use"
        ;;

    # Full pipeline checklists
    checklist-cv)
        test_checklist "CV Screening" \
            "An AI tool that screens CVs and ranks job candidates for recruitment"
        ;;
    checklist-education)
        test_checklist "Education" \
            "An AI system that grades student exams and determines university admissions"
        ;;
    checklist-law-enforcement)
        test_checklist "Law Enforcement" \
            "A predictive policing system that identifies high-crime areas and suggests patrol routes"
        ;;
    checklist-biometrics)
        test_checklist "Biometrics" \
            "A facial recognition system used at airport border control to verify traveler identity"
        ;;

    # Run all quick tests
    all-lookups)
        test_lookup_article "article_3"
        test_lookup_article "article_5"
        test_lookup_article "article_6"
        test_lookup_article "recital_47"
        test_lookup_article "annex_3"
        ;;
    all-classify)
        test_classify "CV Screening" "An AI tool that screens CVs and ranks job candidates for recruitment"
        test_classify "Social Scoring" "A government system that scores citizens based on their social behavior and restricts access to services"
        test_classify "Customer Chatbot" "A customer service chatbot for an e-commerce website"
        test_classify "Spam Filter" "An AI spam filter for email"
        test_classify "Biometric ID" "A facial recognition system used at airport border control to verify traveler identity"
        test_classify "Education AI" "An AI system that grades student exams and determines university admissions"
        test_classify "Predictive Policing" "A predictive policing system that identifies high-crime areas and suggests patrol routes"
        test_classify "Foundation Model" "A large language model trained on internet text, offered as an API for general-purpose use"
        ;;

    *)
        echo "Usage: ./test.sh <test-name>"
        echo ""
        echo "Quick lookups (no LLM, instant):"
        echo "  article-3          Article 3 (Definitions)"
        echo "  article-5          Article 5 (Prohibited practices)"
        echo "  article-6          Article 6 (High-risk classification)"
        echo "  recital-47         Recital 47"
        echo "  annex-3            Annex III"
        echo "  all-lookups        Run all lookups"
        echo ""
        echo "Classification (~2-3 sec each):"
        echo "  cv-screening       HIGH_RISK - employment"
        echo "  social-scoring     PROHIBITED - social scoring"
        echo "  chatbot            LIMITED_RISK - chatbot"
        echo "  spam-filter        MINIMAL_RISK - spam"
        echo "  biometrics         HIGH_RISK - biometrics"
        echo "  education          HIGH_RISK - education"
        echo "  law-enforcement    HIGH_RISK - law enforcement"
        echo "  foundation-model   GPAI - foundation model"
        echo "  all-classify       Run all classifications"
        echo ""
        echo "Prohibited practice checks (~2-3 sec):"
        echo "  emotion-workplace  Emotion recognition at work"
        echo "  facial-scraping    Untargeted facial scraping"
        echo ""
        echo "Full compliance checklists (~10-15 sec each):"
        echo "  checklist-cv              CV screening"
        echo "  checklist-education       Student grading"
        echo "  checklist-law-enforcement Predictive policing"
        echo "  checklist-biometrics      Border control facial recognition"
        exit 1
        ;;
esac
