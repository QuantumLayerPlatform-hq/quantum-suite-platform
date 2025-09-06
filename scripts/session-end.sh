#!/bin/bash

# QLens Session End Script
# Cleans up session and prepares for next session

set -e

# Color codes
BLUE='\033[0;34m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
PURPLE='\033[0;35m'
NC='\033[0m'

print_header() {
    echo -e "${PURPLE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${PURPLE}🏁 QLens Project Session End${NC}"
    echo -e "${PURPLE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

print_section() {
    echo -e "\n${BLUE}📋 $1${NC}"
    echo "────────────────────────────────────────────────"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

# Show session summary
show_session_summary() {
    print_section "Session Summary"
    
    # Get git changes since last session
    LAST_COMMIT_TIME=$(git log -1 --format="%ci" 2>/dev/null || echo "unknown")
    CHANGES_COUNT=$(git status --porcelain 2>/dev/null | wc -l)
    
    echo "Last commit: $LAST_COMMIT_TIME"
    echo "Uncommitted changes: $CHANGES_COUNT files"
    
    if [ "$CHANGES_COUNT" -gt 0 ]; then
        echo ""
        echo "Modified files:"
        git status --short | head -10 | sed 's/^/  /'
    fi
}

# Prompt for progress update
update_progress() {
    print_section "Progress Update"
    
    echo "What was accomplished this session?"
    echo "1. Major feature implementations"
    echo "2. Bug fixes and improvements"  
    echo "3. Configuration changes"
    echo "4. Documentation updates"
    echo "5. Infrastructure work"
    echo ""
    
    read -p "Enter brief description of main accomplishments: " ACCOMPLISHMENTS
    
    if [ -n "$ACCOMPLISHMENTS" ]; then
        # Update PROGRESS.md with new entry
        DATE=$(date "+%Y-%m-%d")
        TIME=$(date "+%H:%M")
        
        # Add to update log in PROGRESS.md
        if [ -f PROGRESS.md ]; then
            # Create backup
            cp PROGRESS.md PROGRESS.md.bak
            
            # Add entry to update log
            sed -i "/## 🔄 Update Log/a | $DATE $TIME | Claude | $ACCOMPLISHMENTS |" PROGRESS.md
            print_success "Progress updated in PROGRESS.md"
        else
            print_warning "PROGRESS.md not found"
        fi
    fi
}

# Check for blockers
identify_blockers() {
    print_section "Blocker Identification"
    
    echo "Are there any blockers for the next session? (y/n)"
    read -p "> " HAS_BLOCKERS
    
    if [ "$HAS_BLOCKERS" = "y" ] || [ "$HAS_BLOCKERS" = "Y" ]; then
        read -p "Describe the main blocker: " BLOCKER_DESCRIPTION
        
        # Add to CLAUDE.md for next session
        if [ -f CLAUDE.md ]; then
            echo "" >> CLAUDE.md
            echo "### **Current Blocker ($(date "+%Y-%m-%d")):**" >> CLAUDE.md
            echo "- $BLOCKER_DESCRIPTION" >> CLAUDE.md
            print_warning "Blocker noted in CLAUDE.md"
        fi
    else
        print_success "No blockers identified"
    fi
}

# Set next session priorities
set_next_priorities() {
    print_section "Next Session Priorities"
    
    echo "What should be the top priority for next session?"
    read -p "Priority 1: " PRIORITY1
    read -p "Priority 2: " PRIORITY2
    read -p "Priority 3: " PRIORITY3
    
    if [ -n "$PRIORITY1" ]; then
        # Update CLAUDE.md with priorities
        if [ -f CLAUDE.md ]; then
            # Remove old priorities section
            sed -i '/### **Next Session Priorities/,$d' CLAUDE.md
            
            # Add new priorities
            echo "" >> CLAUDE.md
            echo "### **Next Session Priorities:**" >> CLAUDE.md
            [ -n "$PRIORITY1" ] && echo "1. 🔥 **P0**: $PRIORITY1" >> CLAUDE.md
            [ -n "$PRIORITY2" ] && echo "2. 🎯 **P1**: $PRIORITY2" >> CLAUDE.md  
            [ -n "$PRIORITY3" ] && echo "3. 🚀 **P2**: $PRIORITY3" >> CLAUDE.md
            
            print_success "Next session priorities set"
        fi
    fi
}

# Commit changes
commit_changes() {
    print_section "Git Commit"
    
    if [ -n "$(git status --porcelain)" ]; then
        echo "Changes detected. Commit them?"
        git status --short
        echo ""
        read -p "Create commit? (y/n): " SHOULD_COMMIT
        
        if [ "$SHOULD_COMMIT" = "y" ] || [ "$SHOULD_COMMIT" = "Y" ]; then
            read -p "Commit message: " COMMIT_MSG
            
            if [ -n "$COMMIT_MSG" ]; then
                git add .
                git commit -m "$COMMIT_MSG

🤖 Generated with [Claude Code](https://claude.ai/code)

Co-Authored-By: Claude <noreply@anthropic.com>"
                
                print_success "Changes committed"
                
                # Ask about pushing
                read -p "Push to remote? (y/n): " SHOULD_PUSH
                if [ "$SHOULD_PUSH" = "y" ] || [ "$SHOULD_PUSH" = "Y" ]; then
                    git push 2>/dev/null && print_success "Changes pushed" || print_warning "Push failed"
                fi
            fi
        fi
    else
        print_info "No changes to commit"
    fi
}

# Service cleanup
cleanup_services() {
    print_section "Service Cleanup"
    
    echo "Clean up running services?"
    echo "1. Keep services running (recommended for quick restart)"
    echo "2. Stop all services (clean shutdown)"
    echo "3. Skip cleanup"
    echo ""
    read -p "Choose option (1-3): " CLEANUP_OPTION
    
    case $CLEANUP_OPTION in
        1)
            print_info "Services left running for next session"
            ;;
        2)
            echo "Stopping QLens services..."
            if command -v make &> /dev/null; then
                make dev-down &> /dev/null && print_success "Services stopped" || print_warning "Some services may still be running"
            else
                kubectl delete -n qlens-staging --all pods &> /dev/null || true
                print_info "Attempted to stop services"
            fi
            ;;
        3)
            print_info "Cleanup skipped"
            ;;
    esac
}

# Generate session report
generate_session_report() {
    print_section "Session Report"
    
    # Create session report
    REPORT_FILE="session-reports/session-$(date "+%Y%m%d-%H%M").md"
    mkdir -p session-reports
    
    cat > "$REPORT_FILE" << EOF
# Claude Code Session Report

**Date:** $(date "+%Y-%m-%d %H:%M:%S")
**Duration:** ~$(echo "scale=1; $(date +%s)/60" | bc 2>/dev/null || echo "unknown") minutes
**Project:** QLens LLM Gateway Service

## Accomplishments
${ACCOMPLISHMENTS:-"No specific accomplishments noted"}

## Files Modified
$(git status --porcelain | head -10)

## Next Session Priorities
${PRIORITY1:+"1. $PRIORITY1"}
${PRIORITY2:+"2. $PRIORITY2"} 
${PRIORITY3:+"3. $PRIORITY3"}

## Current Status
- **Version:** $(cat VERSION 2>/dev/null || echo "unknown")
- **Services:** $(kubectl get pods -n qlens-staging --no-headers 2>/dev/null | wc -l) pods in staging
- **Compilation:** $(go build ./... &> /dev/null && echo "✅ Success" || echo "❌ Issues")

## Context for Next Session
- Review PROGRESS.md for current project status
- Check CLAUDE.md for session continuation context
- Run \`./scripts/session-start.sh\` for environment check

EOF

    print_success "Session report saved to $REPORT_FILE"
}

# Show next session instructions
show_next_session_info() {
    print_section "Next Session Instructions"
    
    echo "To start next Claude Code session:"
    echo ""
    echo "1. Run session startup script:"
    echo "   ${GREEN}./scripts/session-start.sh${NC}"
    echo ""
    echo "2. Review project context:"
    echo "   ${GREEN}cat CLAUDE.md | head -50${NC}"
    echo ""
    echo "3. Check current status:"
    echo "   ${GREEN}cat PROGRESS.md | head -50${NC}"
    echo ""
    echo "4. Get service access info:"
    echo "   ${GREEN}make get-access-info${NC}"
    echo ""
    echo "Key files for context:"
    echo "  📋 PROGRESS.md      - Overall project status"
    echo "  🤖 CLAUDE.md        - Session continuation context"  
    echo "  🔧 Makefile         - All available commands"
    echo "  📊 VERSION          - Current version"
}

# Main execution
main() {
    print_header
    show_session_summary
    
    # Interactive updates
    update_progress
    identify_blockers  
    set_next_priorities
    
    # Cleanup
    commit_changes
    cleanup_services
    generate_session_report
    show_next_session_info
    
    echo ""
    echo -e "${PURPLE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}🎯 Session ended successfully!${NC}"
    echo -e "${PURPLE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

# Run main function
main