#!/usr/bin/env python
# -*- coding: utf-8 -*-

import webapp2


class MainPage(webapp2.RequestHandler):
    def get(self):
        self.response.headers['Content-Type'] = 'text/html'
        self.response.write('こんにちは！')

app = webapp2.WSGIApplication([
    ('/', MainPage),
], debug=True)
