const express = require("express");

const expressMiddleWare = (router) => {
  router.post("/update", function (req, res) {
    res.status(200).send("ok");
  });
};

module.exports = expressMiddleWare;
