#!/usr/bin/env python
# -*- coding: utf-8 -*-

import main
import webtest


def test_get():
    app = webtest.TestApp(main.app)

    response = app.get('/pata?a=cat&b=dog')

    assert response.status_int == 200
    assert response.body.contains('cdaotg')
