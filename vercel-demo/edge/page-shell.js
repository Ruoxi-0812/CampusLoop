(function () {
  'use strict';
  const pages = {
    '/my-listings': ['My listings', 'Manage your items and reservations.'],
    '/cart': ['My listings', 'Manage your items and reservations.'],
    '/login': ['Log in', 'Access your CampusLoop account.'],
    '/signin': ['Log in', 'Access your CampusLoop account.'],
    '/signup': ['Create account', 'Join your campus marketplace.'],
    '/post-item': ['Post an item', 'Find a new home for something you no longer need.'],
    '/messages': ['Messages', 'Coordinate with buyers and sellers.']
  };
  const page = pages[location.pathname];
  if (page) {
    document.getElementById('page-title').textContent = page[0];
    document.getElementById('page-description').textContent = page[1];
  }
}());
