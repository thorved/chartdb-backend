// Auth check script for ChartDB
// This script checks if the user is authenticated via API and redirects to login if not
(function() {
  'use strict';
  
  // Check authentication on page load via API
  async function checkAuth() {
    try {
      const response = await fetch('/sync/api/auth/me', {
        credentials: 'include',
        headers: {
          'Accept': 'application/json'
        }
      });
      
      if (!response.ok) {
        // User is not authenticated, redirect to login
        console.log('Auth check: User not authenticated, redirecting to login...');
        window.location.href = '/sync/login';
        return false;
      }
      return true;
    } catch (err) {
      console.error('Auth check failed:', err);
      window.location.href = '/sync/login';
      return false;
    }
  }
  
  // Run auth check when DOM is ready
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', checkAuth);
  } else {
    checkAuth();
  }
  
  // Also expose the checkAuth function globally for manual checks
  window.checkChartDBAuth = checkAuth;
})();
